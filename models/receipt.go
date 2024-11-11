package models

import (
	"errors"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Receipt struct {
	Retailer     string `json:"retailer"`
	PurchaseDate string `json:"purchaseDate"`
	PurchaseTime string `json:"purchaseTime"`
	Items        []Item `json:"items"`
	Total        string `json:"total"`
}

type Item struct {
	ShortDescription string `json:"shortDescription"`
	Price            string `json:"price"`
}

type ProcessResponse struct {
	ID string `json:"id"`
}

type PointsResponse struct {
	Points int `json:"points"`
}

func (r *Receipt) Validate() error {
	if strings.TrimSpace(r.Retailer) == "" {
		return errors.New("retailer is required")
	}

	if _, err := time.Parse("2006-01-02", r.PurchaseDate); err != nil {
		return errors.New("invalid purchase date format")
	}

	if _, err := time.Parse("15:04", r.PurchaseTime); err != nil {
		return errors.New("invalid purchase time format")
	}

	if len(r.Items) == 0 {
		return errors.New("items are required")
	}

	if _, err := strconv.ParseFloat(r.Total, 64); err != nil {
		return errors.New("invalid total format")
	}

	return nil
}

func (r *Receipt) CalculatePoints() int {
	points := 0

	// rule 1: alphanumeric characters in retailer name
	alphanumeric := regexp.MustCompile(`[a-zA-Z0-9]`)
	points += len(alphanumeric.FindAllString(r.Retailer, -1))

	// rule 2: round dollar amount
	total, _ := strconv.ParseFloat(r.Total, 64)
	if total == float64(int(total)) {
		points += 50
	}

	// rule 3: multiple of 0.25
	if math.Mod(total*100, 25) == 0 {
		points += 25
	}

	// rule 4: every two items
	points += (len(r.Items) / 2) * 5

	// rule 5: description length multiple of 3
	for _, item := range r.Items {
		trimmedLen := len(strings.TrimSpace(item.ShortDescription))
		if trimmedLen%3 == 0 {
			price, _ := strconv.ParseFloat(item.Price, 64)
			points += int(math.Ceil(price * 0.2))
		}
	}

	// rule 6: odd day
	purchaseDate, _ := time.Parse("2006-01-02", r.PurchaseDate)
	if purchaseDate.Day()%2 == 1 {
		points += 6
	}

	// rule 7: time between 2:00 PM and 4:00 PM
	purchaseTime, _ := time.Parse("15:04", r.PurchaseTime)
	targetStart, _ := time.Parse("15:04", "14:00")
	targetEnd, _ := time.Parse("15:04", "16:00")

	if purchaseTime.After(targetStart) && purchaseTime.Before(targetEnd) {
		points += 10
	}

	return points
}
