package spot

type Spot struct {
	ID      int     `json:"id"`
	X       float64 `json:"x"`
	Y       float64 `json:"y"`
	Radius  float64 `json:"radius"`
	TitleEN string  `json:"title_en"`
	BodyEN  string  `json:"body_en"`
	TitleTR string  `json:"title_tr"`
	BodyTR  string  `json:"body_tr"`
	IsNear  bool    `json:"-"`
}
