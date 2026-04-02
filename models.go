package main

import "time"

type Donor struct {
	ID         int64     `json:"id"`
	Phone      string    `json:"phone"`
	Name       string    `json:"name"`
	Email      string    `json:"email"`
	ZipCode    string    `json:"zip_code"`
	LoginCount int       `json:"login_count"`
	CreatedAt  time.Time `json:"created_at"`
}

type Donation struct {
	ID               int64     `json:"id"`
	DonorID          int64     `json:"donor_id"`
	ReceiptNumber    string    `json:"receipt_number"`
	DonationDate     string    `json:"donation_date"`
	Location         string    `json:"location"`
	ItemsDescription string    `json:"items_description"`
	CreatedAt        time.Time `json:"created_at"`
}
