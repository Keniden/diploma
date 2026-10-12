package values

type DriverName struct {
	FirstName  string
	MiddleName string
	LastName   string
}

func (n DriverName) FullName() string {
	name := n.FirstName
	if n.MiddleName != "" {
		name += " " + n.MiddleName
	}

	if n.LastName != "" {
		name += " " + n.LastName
	}

	return name
}

type Driver struct {
	ID         DriverID
	Name       DriverName
	Phone      Phone
	SupplierID SupplierID
}

func NewDriverName(firstName, middleName, lastName string) DriverName {
	return DriverName{
		FirstName:  firstName,
		MiddleName: middleName,
		LastName:   lastName,
	}
}

type Phone string

type Transport struct {
	DriverID     DriverID
	SupplierID   SupplierID
	LicensePlate LicensePlate
}
