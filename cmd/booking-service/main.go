package main

import (
	"log"
	"net/http"

	bookinghttp "github.com/example/coworking/internal/booking/adapters/http"
	"github.com/example/coworking/internal/booking/application"
	busdummy "github.com/example/coworking/internal/booking/infrastructure/bus/dummy"
	"github.com/example/coworking/internal/booking/infrastructure/memory"
	policydummy "github.com/example/coworking/internal/booking/infrastructure/policy/dummy"
)

func main() {
	repo := memory.NewBookingRepository()
	bus := busdummy.NewEventBus()
	availabilityChecker := policydummy.NewAvailabilityChecker()
	priceCalculator := policydummy.NewPriceCalculator()
	svc := application.NewService(repo, bus, availabilityChecker, priceCalculator)
	handler := bookinghttp.NewBookingHandler(svc)

	log.Println("starting booking service on :8080")
	if err := http.ListenAndServe(":8080", handler); err != nil {
		log.Fatal(err)
	}
}
