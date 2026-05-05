package ceroflake

import "fmt"

// EntityType identifies the kind of domain object an ID belongs to.
// Values 1–15 are available; 0 means generic / untagged.
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
	// 8–15 reserved for caller extension
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

func (e EntityType) Validate() error {
	if e > 15 {
		return fmt.Errorf("ceroflake: entity type %d exceeds max value 15", uint8(e))
	}
	return nil
}
