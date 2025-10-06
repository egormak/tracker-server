package entity

type PlanPercents struct {
	Title         string   `json:"title"`
	Date          string   `json:"date"`
	CurrentChoice int      `json:"current_choice"`
	Plans         []string `json:"plans"`
	Plan          []int    `json:"plan"`
	Work          []int    `json:"work"`
	Learn         []int    `json:"learn"`
	Rest          []int    `json:"rest"`
}
