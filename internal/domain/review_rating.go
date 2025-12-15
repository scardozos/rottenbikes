package domain

type RatingSubcategory string

const (
	RatingSubcategoryOverall    RatingSubcategory = "overall"
	RatingSubcategoryBreaks     RatingSubcategory = "breaks"
	RatingSubcategorySeat       RatingSubcategory = "seat"
	RatingSubcategorySturdiness RatingSubcategory = "sturdiness"
	RatingSubcategoryPower      RatingSubcategory = "power"
	RatingSubcategoryPedals     RatingSubcategory = "pedals"
)

type ReviewRating struct {
	ReviewID    int64             `db:"review_id"`   // PK
	Subcategory RatingSubcategory `db:"subcategory"` // PK
	Score       int16             `db:"score"`       // 1–5
}
