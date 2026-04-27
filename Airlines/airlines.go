package main

import (
	"fmt"
	"strings"
)

type Plane struct {
	ID       int
	Model    string
	Capacity int
}

type Passenger struct {
	ID   int
	Name string
}

type Flight struct {
	ID           int
	Number       string
	Plane        Plane
	From         string
	To           string
	Rezervations map[int]*Rezervation
}

type Rezervation struct {
	ID        int
	Passenger Passenger
	Flight    *Flight
}

type RezervationSystem struct {
	Flights      []*Flight
	Rezervations []*Rezervation
	nextResId    int
}

type Stringer interface {
	String() string
}

type FlightSearcher interface {
	SearchFrom(port string) []*Flight
	SearchTo(port string) []*Flight
}

type ReservationFinder interface {
	FindReservationsForPassenger(passengerID int) []*Rezervation
}

func (f Flight) String() string {
	taken := len(f.Rezervations)
	free := f.Plane.Capacity - taken
	return fmt.Sprintf("Flight %s: %s -> %s | Plane: %s | Free seats: %d/%d",
		f.Number, f.From, f.To, f.Plane.Model, free, f.Plane.Capacity)
}

func (f *Flight) Reserve(r Rezervation) bool {
	if len(f.Rezervations) >= f.Plane.Capacity {
		fmt.Println("No free seats on flight", f.Number)
		return false
	}
	if _, exists := f.Rezervations[r.Passenger.ID]; exists {
		fmt.Printf("Passenger %s already has a reservation on flight %s\n", r.Passenger.Name, f.Number)
		return false
	}
	f.Rezervations[r.Passenger.ID] = &r
	fmt.Printf("Reserved: %s -> flight %s\n", r.Passenger.Name, f.Number)
	return true
}

func (f *Flight) CancelReservation(passengerID int) bool {
	if _, exists := f.Rezervations[passengerID]; !exists {
		fmt.Println("Nie znaleziono rezerwacji do anulowania")
		return false
	}
	delete(f.Rezervations, passengerID)
	fmt.Printf("Anulowano rezerwację pasażera ID=%d na lot %s\n", passengerID, f.Number)
	return true
}

func (f *Flight) FreeSeats() int {
	return f.Plane.Capacity - len(f.Rezervations)
}

func NewSystem() *RezervationSystem {
	return &RezervationSystem{
		Flights:      make([]*Flight, 0),
		Rezervations: make([]*Rezervation, 0),
		nextResId:    1,
	}
}

func (s *RezervationSystem) AddFlight(flight *Flight) {
	s.Flights = append(s.Flights, flight)
}

func (s *RezervationSystem) Reserve(passenger Passenger, flight *Flight) bool {
	r := Rezervation{
		ID:        s.nextResId,
		Passenger: passenger,
		Flight:    flight,
	}
	if flight.Reserve(r) {
		s.Rezervations = append(s.Rezervations, &r)
		s.nextResId++
		return true
	}
	return false
}

func (s *RezervationSystem) CancelReservation(passengerID int, flight *Flight) bool {
	nowe := make([]*Rezervation, 0)
	for _, r := range s.Rezervations {
		if r.Passenger.ID != passengerID || r.Flight != flight {
			nowe = append(nowe, r)
		}
	}
	if len(nowe) != len(s.Rezervations) {
		s.Rezervations = nowe
		return true
	}
	return false
}

func (s *RezervationSystem) FindFlightsFrom(port string) []*Flight {
	wynik := make([]*Flight, 0)
	for _, f := range s.Flights {
		if strings.EqualFold(f.From, port) {
			wynik = append(wynik, f)
		}
	}
	return wynik
}

func (s *RezervationSystem) FindFlightsTo(port string) []*Flight {
	wynik := make([]*Flight, 0)
	for _, f := range s.Flights {
		if strings.EqualFold(f.To, port) {
			wynik = append(wynik, f)
		}
	}
	return wynik
}

func (s *RezervationSystem) FindReservationsForPassenger(passengerID int) []*Rezervation {
	wynik := make([]*Rezervation, 0)
	for _, r := range s.Rezervations {
		if r.Passenger.ID == passengerID {
			wynik = append(wynik, r)
		}
	}
	return wynik
}

func main() {
	system := NewSystem()

	s1 := Plane{ID: 1, Model: "Boeing 737", Capacity: 3}
	s2 := Plane{ID: 2, Model: "Airbus A320", Capacity: 2}

	flight1 := &Flight{ID: 1, Number: "LO101", Plane: s1, From: "WAW", To: "JFK", Rezervations: make(map[int]*Rezervation)}
	flight2 := &Flight{ID: 2, Number: "LO102", Plane: s2, From: "KRK", To: "WAW", Rezervations: make(map[int]*Rezervation)}
	flight3 := &Flight{ID: 3, Number: "LO103", Plane: s1, From: "WAW", To: "CDG", Rezervations: make(map[int]*Rezervation)}

	system.AddFlight(flight1)
	system.AddFlight(flight2)
	system.AddFlight(flight3)

	p1 := Passenger{ID: 1, Name: "Jan Kowalski"}
	p2 := Passenger{ID: 2, Name: "Anna Nowak"}
	p3 := Passenger{ID: 3, Name: "Piotr Wiśnia"}

	fmt.Println("Flights: ")
	fmt.Println(flight1.String())
	fmt.Println(flight2.String())
	fmt.Println(flight3.String())

	fmt.Println("\nReservations: ")
	system.Reserve(p1, flight1) // OK
	system.Reserve(p2, flight1) // OK
	system.Reserve(p3, flight1) // OK (full)
	system.Reserve(p1, flight1) // ERROR (duplicate)
	system.Reserve(p1, flight2) // OK

	fmt.Println("\nReservations after: ")
	fmt.Println(flight1.String())
	fmt.Println(flight2.String())

	fmt.Println("\nFree seats: ")
	fmt.Printf("Flight %s: %d free seats\n", flight1.Number, flight1.FreeSeats())
	fmt.Printf("Flight %s: %d free seats\n", flight2.Number, flight2.FreeSeats())

	fmt.Println("\nCancel reservation for Anna Nowak")
	system.CancelReservation(p2.ID, flight1)

	fmt.Println("\nRezervations for Anna Nowak:")
	for _, r := range system.FindReservationsForPassenger(p2.ID) {
		fmt.Printf("- %s for flight %s\n", r.Passenger.Name, r.Flight.Number)
	}

	fmt.Println("\nRezervations for Jan Kowalski:")
	for _, r := range system.FindReservationsForPassenger(1) {
		fmt.Printf("- %s for flight %s\n", r.Passenger.Name, r.Flight.Number)
	}

	fmt.Println("\nFlights from WAW:")
	for _, l := range system.FindFlightsFrom("WAW") {
		fmt.Println(l.String())
	}

	fmt.Println("\nFlights to WAW:")
	for _, l := range system.FindFlightsTo("WAW") {
		fmt.Println(l.String())
	}
}
