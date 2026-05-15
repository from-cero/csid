package crid

import "fmt"

// EntityType identifies the domain object an ID belongs to.
// Values 0–15 fit the default 4-bit EntityBits layout.
type EntityType uint8

const (
	EntityGeneric  EntityType = 0
	EntityUser     EntityType = 1
	EntityOrder    EntityType = 2
	EntityProduct  EntityType = 3
	EntityPayment  EntityType = 4
	EntityInvoice  EntityType = 5
	EntityShipment EntityType = 6
	EntitySession  EntityType = 7
	// 8–15 available for caller-defined types
)

var entityNames = map[EntityType]string{
	EntityGeneric:  "generic",
	EntityUser:     "user",
	EntityOrder:    "order",
	EntityProduct:  "product",
	EntityPayment:  "payment",
	EntityInvoice:  "invoice",
	EntityShipment: "shipment",
	EntitySession:  "session",
}

func (e EntityType) String() string {
	if name, ok := entityNames[e]; ok {
		return name
	}
	return fmt.Sprintf("entity(%d)", uint8(e))
}
