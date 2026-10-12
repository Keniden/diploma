package entities_test

import (
	"testing"

	"last-mile-task/internal/shipping/domain/entities"
	"last-mile-task/internal/shipping/domain/values"
)

func TestNewShipment(t *testing.T) {
	s := entities.NewShipment()

	if s.ID == values.NilID {
		t.Fatal("expected non-nil ID")
	}
	if s.ShortID == 0 {
		t.Fatal("expected non-zero ShortID")
	}
	if s.Status != values.ShipmentStatusNew {
		t.Fatalf("expected status NEW, got %v", s.Status)
	}
}

func TestNewShipment_ShortIDIsUnique(t *testing.T) {
	const n = 100
	seen := make(map[values.ShipmentShortID]bool)
	for range n {
		s := entities.NewShipment()
		if seen[s.ShortID] {
			t.Fatalf("duplicate short ID: %d", s.ShortID)
		}
		seen[s.ShortID] = true
	}
}

func TestShipment_AssignDriver(t *testing.T) {
	s := entities.NewShipment()
	driverID := values.NewID()
	var supplierID values.SupplierID = 42

	err := s.AssignDriver(driverID, supplierID)
	if err != nil {
		t.Fatal(err)
	}

	if s.DriverID != driverID {
		t.Fatalf("expected driver ID %v, got %v", driverID, s.DriverID)
	}
	if s.SupplierID != supplierID {
		t.Fatalf("expected supplier ID %d, got %d", supplierID, s.SupplierID)
	}
	if s.Status != values.ShipmentStatusAssigned {
		t.Fatalf("expected status ASSIGNED, got %v", s.Status)
	}
}

func TestShipment_AssignDriver_AlreadyAssigned(t *testing.T) {
	s := entities.NewShipment()
	driverID := values.NewID()

	if err := s.AssignDriver(driverID, 42); err != nil {
		t.Fatal(err)
	}

	// second assign should fail
	err := s.AssignDriver(values.NewID(), 99)
	if err == nil {
		t.Fatal("expected error on second driver assignment")
	}
	// driver and supplier should remain unchanged
	if s.DriverID != driverID {
		t.Fatal("driver ID should not change after failed assign")
	}
	if s.SupplierID != 42 {
		t.Fatal("supplier ID should not change after failed assign")
	}
}

func TestShipment_SetLicensePlate(t *testing.T) {
	s := entities.NewShipment()
	plate := values.LicensePlate("A123BC")

	err := s.SetLicensePLate(plate)
	if err != nil {
		t.Fatal(err)
	}

	if s.LicensePlate != plate {
		t.Fatalf("expected license plate %q, got %q", plate, s.LicensePlate)
	}
}

func TestShipment_IsAssigned(t *testing.T) {
	s := entities.NewShipment()
	if s.IsAssigned() {
		t.Fatal("expected not assigned for new shipment")
	}

	if err := s.AssignDriver(values.NewID(), 42); err != nil {
		t.Fatal(err)
	}

	if !s.IsAssigned() {
		t.Fatal("expected assigned after driver assignment")
	}
}

func TestShipment_IsReserveLoading(t *testing.T) {
	s := entities.NewShipment()
	if s.IsReserveLoading() {
		t.Fatal("expected not reserve loading for new shipment")
	}

	_ = s.AssignDriver(values.NewID(), 42)
	_ = s.ReserveLoading()

	if !s.IsReserveLoading() {
		t.Fatal("expected reserve loading after calling ReserveLoading")
	}
}

func TestShipment_IsReserveUnloading(t *testing.T) {
	s := entities.NewShipment()
	if s.IsReserveUnloading() {
		t.Fatal("expected not reserve unloading for new shipment")
	}

	_ = s.AssignDriver(values.NewID(), 42)
	_ = s.ReserveLoading()
	_ = s.Loading()
	_ = s.Start()
	_ = s.ReserveUnloading()

	if !s.IsReserveUnloading() {
		t.Fatal("expected reserve unloading after calling ReserveUnloading")
	}
}

func TestShipment_IsUnloading(t *testing.T) {
	s := entities.NewShipment()
	if s.IsUnloading() {
		t.Fatal("expected not unloading for new shipment")
	}

	_ = s.AssignDriver(values.NewID(), 42)
	_ = s.ReserveLoading()
	_ = s.Loading()
	_ = s.Start()
	_ = s.ReserveUnloading()
	_ = s.Unloading()

	if !s.IsUnloading() {
		t.Fatal("expected unloading after calling Unloading")
	}
}

func TestShipment_ValidTransitions(t *testing.T) {
	tests := []struct {
		name   string
		action func(s *entities.Shipment) error
		want   values.ShipmentStatus
	}{
		{
			name:   "New -> Assigned",
			action: func(s *entities.Shipment) error { return s.AssignDriver(values.NewID(), 42) },
			want:   values.ShipmentStatusAssigned,
		},
		{
			name:   "New -> Canceled",
			action: func(s *entities.Shipment) error { return s.Cancel() },
			want:   values.ShipmentStatusCanceled,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := entities.NewShipment()
			if err := tt.action(s); err != nil {
				t.Fatal(err)
			}
			if s.Status != tt.want {
				t.Fatalf("expected status %v, got %v", tt.want, s.Status)
			}
		})
	}

	// Test chained transitions that require prior state
	t.Run("Assigned -> ReserveLoading", func(t *testing.T) {
		s := entities.NewShipment()
		if err := s.AssignDriver(values.NewID(), 42); err != nil {
			t.Fatal(err)
		}
		if err := s.ReserveLoading(); err != nil {
			t.Fatal(err)
		}
		if s.Status != values.ShipmentStatusReserveLoading {
			t.Fatalf("expected RESERVE_LOADING, got %v", s.Status)
		}
	})

	t.Run("ReserveLoading -> Loading", func(t *testing.T) {
		s := entities.NewShipment()
		if err := s.AssignDriver(values.NewID(), 42); err != nil {
			t.Fatal(err)
		}
		if err := s.ReserveLoading(); err != nil {
			t.Fatal(err)
		}
		if err := s.Loading(); err != nil {
			t.Fatal(err)
		}
		if s.Status != values.ShipmentStatusLoading {
			t.Fatalf("expected LOADING, got %v", s.Status)
		}
	})

	t.Run("Assigned -> Canceled", func(t *testing.T) {
		s := entities.NewShipment()
		if err := s.AssignDriver(values.NewID(), 42); err != nil {
			t.Fatal(err)
		}
		if err := s.Cancel(); err != nil {
			t.Fatal(err)
		}
		if s.Status != values.ShipmentStatusCanceled {
			t.Fatalf("expected CANCELED, got %v", s.Status)
		}
	})

	t.Run("Loading -> InProgress", func(t *testing.T) {
		s := entities.NewShipment()
		if err := s.AssignDriver(values.NewID(), 42); err != nil {
			t.Fatal(err)
		}
		if err := s.ReserveLoading(); err != nil {
			t.Fatal(err)
		}
		if err := s.Loading(); err != nil {
			t.Fatal(err)
		}
		if err := s.Start(); err != nil {
			t.Fatal(err)
		}
		if s.Status != values.ShipmentStatusInProgress {
			t.Fatalf("expected IN_PROGRESS, got %v", s.Status)
		}
	})

	t.Run("Loading -> Canceled", func(t *testing.T) {
		s := entities.NewShipment()
		if err := s.AssignDriver(values.NewID(), 42); err != nil {
			t.Fatal(err)
		}
		if err := s.ReserveLoading(); err != nil {
			t.Fatal(err)
		}
		if err := s.Loading(); err != nil {
			t.Fatal(err)
		}
		if err := s.Cancel(); err != nil {
			t.Fatal(err)
		}
		if s.Status != values.ShipmentStatusCanceled {
			t.Fatalf("expected CANCELED, got %v", s.Status)
		}
	})

	t.Run("InProgress -> Completed", func(t *testing.T) {
		s := entities.NewShipment()
		if err := s.AssignDriver(values.NewID(), 42); err != nil {
			t.Fatal(err)
		}
		if err := s.ReserveLoading(); err != nil {
			t.Fatal(err)
		}
		if err := s.Loading(); err != nil {
			t.Fatal(err)
		}
		if err := s.Start(); err != nil {
			t.Fatal(err)
		}
		if err := s.Complete(); err != nil {
			t.Fatal(err)
		}
		if s.Status != values.ShipmentStatusCompleted {
			t.Fatalf("expected COMPLETED, got %v", s.Status)
		}
	})

	t.Run("InProgress -> ReserveUnloading", func(t *testing.T) {
		s := entities.NewShipment()
		if err := s.AssignDriver(values.NewID(), 42); err != nil {
			t.Fatal(err)
		}
		if err := s.ReserveLoading(); err != nil {
			t.Fatal(err)
		}
		if err := s.Loading(); err != nil {
			t.Fatal(err)
		}
		if err := s.Start(); err != nil {
			t.Fatal(err)
		}
		if err := s.ReserveUnloading(); err != nil {
			t.Fatal(err)
		}
		if s.Status != values.ShipmentStatusReserveUnloading {
			t.Fatalf("expected RESERVE_UNLOADING, got %v", s.Status)
		}
	})

	t.Run("ReserveUnloading -> Unloading", func(t *testing.T) {
		s := entities.NewShipment()
		if err := s.AssignDriver(values.NewID(), 42); err != nil {
			t.Fatal(err)
		}
		if err := s.ReserveLoading(); err != nil {
			t.Fatal(err)
		}
		if err := s.Loading(); err != nil {
			t.Fatal(err)
		}
		if err := s.Start(); err != nil {
			t.Fatal(err)
		}
		if err := s.ReserveUnloading(); err != nil {
			t.Fatal(err)
		}
		if err := s.Unloading(); err != nil {
			t.Fatal(err)
		}
		if s.Status != values.ShipmentStatusUnloading {
			t.Fatalf("expected UNLOADING, got %v", s.Status)
		}
	})

	t.Run("Unloading -> Completed", func(t *testing.T) {
		s := entities.NewShipment()
		if err := s.AssignDriver(values.NewID(), 42); err != nil {
			t.Fatal(err)
		}
		if err := s.ReserveLoading(); err != nil {
			t.Fatal(err)
		}
		if err := s.Loading(); err != nil {
			t.Fatal(err)
		}
		if err := s.Start(); err != nil {
			t.Fatal(err)
		}
		if err := s.ReserveUnloading(); err != nil {
			t.Fatal(err)
		}
		if err := s.Unloading(); err != nil {
			t.Fatal(err)
		}
		if err := s.Complete(); err != nil {
			t.Fatal(err)
		}
		if s.Status != values.ShipmentStatusCompleted {
			t.Fatalf("expected COMPLETED, got %v", s.Status)
		}
	})

	t.Run("InProgress -> Canceled", func(t *testing.T) {
		s := entities.NewShipment()
		if err := s.AssignDriver(values.NewID(), 42); err != nil {
			t.Fatal(err)
		}
		if err := s.ReserveLoading(); err != nil {
			t.Fatal(err)
		}
		if err := s.Loading(); err != nil {
			t.Fatal(err)
		}
		if err := s.Start(); err != nil {
			t.Fatal(err)
		}
		if err := s.Cancel(); err != nil {
			t.Fatal(err)
		}
		if s.Status != values.ShipmentStatusCanceled {
			t.Fatalf("expected CANCELED, got %v", s.Status)
		}
	})
}

func TestShipment_InvalidTransitions(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(s *entities.Shipment)
		action    func(s *entities.Shipment) error
		startFrom values.ShipmentStatus
	}{
		{
			name:      "New -> Loading",
			setup:     nil,
			action:    func(s *entities.Shipment) error { return s.Loading() },
			startFrom: values.ShipmentStatusNew,
		},
		{
			name:      "New -> InProgress",
			setup:     nil,
			action:    func(s *entities.Shipment) error { return s.Start() },
			startFrom: values.ShipmentStatusNew,
		},
		{
			name:      "New -> Completed",
			setup:     nil,
			action:    func(s *entities.Shipment) error { return s.Complete() },
			startFrom: values.ShipmentStatusNew,
		},
		{
			name:      "New -> ReserveLoading",
			setup:     nil,
			action:    func(s *entities.Shipment) error { return s.ReserveLoading() },
			startFrom: values.ShipmentStatusNew,
		},
		{
			name:      "New -> Unloading",
			setup:     nil,
			action:    func(s *entities.Shipment) error { return s.Unloading() },
			startFrom: values.ShipmentStatusNew,
		},
		{
			name: "Assigned -> InProgress",
			setup: func(s *entities.Shipment) {
				_ = s.AssignDriver(values.NewID(), 42)
			},
			action:    func(s *entities.Shipment) error { return s.Start() },
			startFrom: values.ShipmentStatusAssigned,
		},
		{
			name: "Assigned -> Completed",
			setup: func(s *entities.Shipment) {
				_ = s.AssignDriver(values.NewID(), 42)
			},
			action:    func(s *entities.Shipment) error { return s.Complete() },
			startFrom: values.ShipmentStatusAssigned,
		},
		{
			name: "Assigned -> Unloading",
			setup: func(s *entities.Shipment) {
				_ = s.AssignDriver(values.NewID(), 42)
			},
			action:    func(s *entities.Shipment) error { return s.Unloading() },
			startFrom: values.ShipmentStatusAssigned,
		},
		{
			name: "Completed -> Canceled",
			setup: func(s *entities.Shipment) {
				_ = s.AssignDriver(values.NewID(), 42)
				_ = s.ReserveLoading()
				_ = s.Loading()
				_ = s.Start()
				_ = s.ReserveUnloading()
				_ = s.Unloading()
				_ = s.Complete()
			},
			action:    func(s *entities.Shipment) error { return s.Cancel() },
			startFrom: values.ShipmentStatusCompleted,
		},
		{
			name: "Canceled -> Assigned",
			setup: func(s *entities.Shipment) {
				_ = s.Cancel()
			},
			action:    func(s *entities.Shipment) error { return s.AssignDriver(values.NewID(), 42) },
			startFrom: values.ShipmentStatusCanceled,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := entities.NewShipment()
			if tt.setup != nil {
				tt.setup(s)
			}
			if s.Status != tt.startFrom {
				t.Fatalf("setup failed: expected status %v, got %v", tt.startFrom, s.Status)
			}

			err := tt.action(s)
			if err == nil {
				t.Fatal("expected error for invalid transition")
			}
			if s.Status != tt.startFrom {
				t.Fatalf("status changed after invalid transition: was %v, now %v", tt.startFrom, s.Status)
			}
		})
	}
}

func TestShipment_RevertReserveTransitions(t *testing.T) {
	t.Run("ReserveLoading -> Assigned", func(t *testing.T) {
		s := entities.NewShipment()
		if err := s.AssignDriver(values.NewID(), 42); err != nil {
			t.Fatal(err)
		}
		if err := s.ReserveLoading(); err != nil {
			t.Fatal(err)
		}
		if err := s.RevertReserveLoading(); err != nil {
			t.Fatal(err)
		}
		if s.Status != values.ShipmentStatusAssigned {
			t.Fatalf("expected ASSIGNED after revert, got %v", s.Status)
		}
	})

	t.Run("ReserveUnloading -> InProgress", func(t *testing.T) {
		s := entities.NewShipment()
		if err := s.AssignDriver(values.NewID(), 42); err != nil {
			t.Fatal(err)
		}
		if err := s.ReserveLoading(); err != nil {
			t.Fatal(err)
		}
		if err := s.Loading(); err != nil {
			t.Fatal(err)
		}
		if err := s.Start(); err != nil {
			t.Fatal(err)
		}
		if err := s.ReserveUnloading(); err != nil {
			t.Fatal(err)
		}
		if err := s.RevertReserveUnloading(); err != nil {
			t.Fatal(err)
		}
		if s.Status != values.ShipmentStatusInProgress {
			t.Fatalf("expected IN_PROGRESS after revert, got %v", s.Status)
		}
	})

	t.Run("Revert from non-reserve status fails", func(t *testing.T) {
		s := entities.NewShipment()
		if err := s.AssignDriver(values.NewID(), 42); err != nil {
			t.Fatal(err)
		}
		if err := s.Loading(); err != nil {
			t.Fatal(err)
		}
		if err := s.RevertReserveLoading(); err == nil {
			t.Fatal("expected error when reverting from a non-reserve status")
		}
	})
}

func TestShipment_FullLifecycle(t *testing.T) {
	s := entities.NewShipment()
	driverID := values.NewID()

	if err := s.AssignDriver(driverID, 42); err != nil {
		t.Fatal(err)
	}
	if err := s.SetLicensePLate("A123BC"); err != nil {
		t.Fatal(err)
	}
	if err := s.ReserveLoading(); err != nil {
		t.Fatal(err)
	}
	if err := s.Loading(); err != nil {
		t.Fatal(err)
	}
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	if err := s.ReserveUnloading(); err != nil {
		t.Fatal(err)
	}
	if err := s.Unloading(); err != nil {
		t.Fatal(err)
	}
	if err := s.Complete(); err != nil {
		t.Fatal(err)
	}

	if s.Status != values.ShipmentStatusCompleted {
		t.Fatalf("expected COMPLETED, got %v", s.Status)
	}
	if s.DriverID != driverID {
		t.Fatal("driver ID preserved through lifecycle")
	}
	if s.LicensePlate != "A123BC" {
		t.Fatal("license plate preserved through lifecycle")
	}
}
