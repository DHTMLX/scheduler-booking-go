package data

type Doctor struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Subtitle string `json:"subtitle"`
	Details  string `json:"details"`
	Category string `json:"category"`
	Price    string `json:"price"`
	Gap      int    `json:"gap"`
	SlotSize int    `json:"slot_size"`
	ImageURL string `json:"-"`

	DoctorSchedule []DoctorSchedule `json:"-"`
	OccupiedSlots  []OccupiedSlot   `json:"-"`
	Review         Review           `json:"-" gorm:"foreignkey:DoctorID"`
}

type Review struct {
	ID       int `json:"-"`
	Count    int `json:"count"`
	Stars    int `json:"stars"`
	DoctorID int `json:"-"`
}

type DoctorSchedule struct {
	ID               int
	DoctorID         int
	From             int
	To               int
	Date             int64
	Rrule            string
	RecurringEventID string
	OriginalStart    string
	Duration         int // in seconds
	Deleted          bool
}

type OccupiedSlot struct {
	ID            int    `json:"id"`
	DoctorID      int    `json:"doctor_id"`
	Date          int64  `json:"date"`
	ClientName    string `json:"client_name"`
	ClientEmail   string `json:"client_email"`
	ClientDetails string `json:"client_details"`
}
