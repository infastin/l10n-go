// Code generated by l10n-go; DO NOT EDIT.

package l10n

import (
	"strings"
	"fmt"
)

type en_Localizer struct{}

func (en_l en_Localizer) BankAccount(money float64) string {
	b0 := new(strings.Builder)

	b0.WriteString("You have $")
	fmt.Fprintf(b0, "%+.3f", money)
	b0.WriteString(" dollars in your bank account.")

	return b0.String()
}