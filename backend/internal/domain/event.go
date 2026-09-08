type Event struct {
	ID                        uuid.UUID
	OrganizerID               uuid.UUID
	CategoryID                uuid.UUID
	VenueID                   uuid.UUID
	Title                     string
	Description               string
	EventDate                 time.Time
	StartTime                 time.Time
	EndTime                   time.Time
	Banner                    string
	Language                  string
	AgeRestriction            int
	CancellationAllowed       bool
	CancellationDeadlineHours int
	SeatLayoutType            string
	BookingLimitPerUser       int
	Status                    string
	CreatedAt                 time.Time
	UpdatedAt                 time.Time
}