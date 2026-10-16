package location

type Coordinates struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

func (c Coordinates) IsValid() bool {
	return c.Latitude >= -90 &&
		c.Latitude <= 90 &&
		c.Longitude >= -180 &&
		c.Longitude <= 180
}
