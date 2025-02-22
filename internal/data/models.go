package data

import (
	"database/sql"
	"errors"
)

var ErrRecordNotFound = errors.New("record not found")
var ErrDuplicateEmail = errors.New("duplicate email")

type Models struct {
	Creditcard CreditCardModel
	Users      UserModel
	Product    ProductModel
	Orders     OrdersModel
}

func NewModel(db *sql.DB) Models {
	return Models{
		Creditcard: CreditCardModel{db},
		Users:      UserModel{db},
		Product:    ProductModel{db},
		Orders:     OrdersModel{db},
	}
}
