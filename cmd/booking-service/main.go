package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	bookinghttp "github.com/example/coworking/internal/booking/adapters/http"
	"github.com/example/coworking/internal/booking/application"
	busdummy "github.com/example/coworking/internal/booking/infrastructure/bus/dummy"
	"github.com/example/coworking/internal/booking/infrastructure/memory"
	"github.com/example/coworking/internal/booking/infrastructure/outbox"
	policydummy "github.com/example/coworking/internal/booking/infrastructure/policy/dummy"
	"github.com/example/coworking/internal/booking/infrastructure/transaction"
	"github.com/example/coworking/internal/config"
)

func main() {
	//создаем cfg. вызываем функцию Load() которая загружает конф приложения (порт сервера, уровень лог и тд)
	cfg, err := config.Load()
	if err != nil { //если произошла ошибка при загрузке конфигов 
		slog.Error("failed to load config", "error", err)//выводим сообщение об ошибке
		os.Exit(1)//отправляем ОС код завершения 1 (программа завершилась с ошибкой)
	}

	// Set up structured logging.
	var logLevel slog.Level //переменная logLevel будет хранить уровень логирования 
	switch cfg.LogLevel { //проверяем, что пришло из cfg
	case "debug": //если debug
		logLevel = slog.LevelDebug //присваиваем переменной logLevel уровень Debug 
	case "warn": //если warm 
		logLevel = slog.LevelWarn
	case "error": //если error 
		logLevel = slog.LevelError
	default: //если значение не указано или другое 
		logLevel = slog.LevelInfo //используем инфо по умолчанию 
	}
	//создаем новый логгер 
	//NewJSONHandler обработчик логов будет выводить логи в формате JSON
	//os.Stdout - вывод в консоль 
	//указатель на структуру HandlerOptions передаем уровни логирования 
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel}))
	//новый логгер устанавливаем по умолчанию 
	slog.SetDefault(logger)

	// Wire dependencies.
	//создаем переменную repo и вызываем конструктор NewBookingRepository() и пакета memory 
	//этот репо будет сохранять и получать бронирования 
	repo := memory.NewBookingRepository()
	bus := busdummy.NewEventBus() //отвечает за отправку событий в другие сервисы 
	availabilityChecker := policydummy.NewAvailabilityChecker(repo)//проверка доступности комнаты и дат
	priceCalculator := policydummy.NewPriceCalculator() //расчет стоимости бронирования 
	eventStore := outbox.NewEventStore(bus) //передаем перменную (bus). сохраням доменные события, после сохранения события отправляются 
	uow := transaction.NewUnitOfWork(repo, eventStore) //управление бизнес операцией 

	//создаем сервис приложения svc 
	//вызываем конструктор NewService и передаем все зависимости 
	svc := application.NewService(repo, bus, availabilityChecker, priceCalculator, uow, logger)
	//конструктор NewRouter ренистирует эндпоинтсы 
	handler := bookinghttp.NewRouter(svc, logger)

	// Create HTTP server.
	srv := &http.Server{
		Addr:         ":" + cfg.AppPort, //адрес на котором будет работать http сервер 
		Handler:      handler, //передаем роутер, который создали выше, для отправки запросов на нужные эндпоинтсы
		ReadTimeout:  10 * time.Second, //макс время чтения http запроса 
		WriteTimeout: 10 * time.Second,//макс время зависи ответа клиенту 
		IdleTimeout:  60 * time.Second, //сколько времени держать соед. открытым 
	}

	// Graceful shutdown on SIGINT / SIGTERM.
	//signal.NotifyContext слушает сигналы ОС 
	//os.Interrupt сигнал когда пользователь нажимает  Ctrl + C
	//syscall.SIGTERM сигнал остановки процесса 
	//когда приходит один из этих сигналов, контекст ctx будет отменен 
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	//stop() освоброждает ресурсы. выполнится после завершения main 
	defer stop()

	go func() { //запуск анонимной горутины  
		//записываем в лог сообщение, что сервер запускается 
		logger.Info("starting booking service", "addr", srv.Addr)
		//метод ListenAndServe() запускает http сервер 
		//начинает слушать порт и принимать запросы 
		//если при запуске произошла ошибка и эта ошибка не http.ErrServerClosed
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			//значит сервер запустился с реальной ошибкой. Записываем ошибку в лог 
			logger.Error("server error", "error", err)
			os.Exit(1) //завершаем программу с кодом ошибки 
		}
	}()

	<-ctx.Done() //ждем пока контекст не будет отменен 
	logger.Info("shutting down gracefully...")

	//создаем новый контекст shutdownCtx
	//context.WithTimeout будет активен 10 секунд 
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()//освобождает ресурсы контекста 

	//вызываем метод Shutdown. Сервер перестает принимать новые запросы 
	//но завершает уже выполняющиеся 
	if err := srv.Shutdown(shutdownCtx); err != nil {
		//если при завершении произошла ошибка, записываем ее в лог 
		logger.Error("shutdown error", "error", err)
		os.Exit(1) //и завершаем программу 
	}
	logger.Info("server stopped") //пишем в лог, что сервер остановлен 
}
