package wms

type Driver struct {
	Name  string
	Phone string
}

type Shipment struct {
	Driver Driver
}
