package extract

import (
	"testing"

	"github.com/danielededo/cedolo/internal/payslip"
	"github.com/danielededo/cedolo/internal/synth"
)

// TestPersonAndEmployer covers the codes and names of ADR 0010: the name is
// the words the codice fiscale encodes, here after a code and before a date,
// with an apostrophe and two given names; the employer's code is the one on
// a line that says what it is, and its name ends with the company form.
func TestPersonAndEmployer(t *testing.T) {
	cf := synth.CodiceFiscale("D'ANGELO MARIA GRAZIA")
	vat := synth.VATNumber("0123456789")
	other := synth.VATNumber("9876543210")
	pg := page(
		[]string{"40:Ditta"},
		[]string{"40:ACME ESEMPIO S.r.l. con socio unico", "400:Matricola 12345678901"},
		[]string{"40:Cod. fiscale: " + vat, "300:Rif. " + other},
		[]string{"40:Codice", "100:Cognome e nome", "300:Data assunzione"},
		[]string{"40:136", "100:D'ANGELO MARIA GRAZIA", "300:12/12/22"},
		[]string{"40:Codice fiscale", "160:" + cf},
	)
	lines := [][]line{buildLines(pg)}
	if got := person(lines); got != (payslip.Party{ID: cf, Name: "Maria Grazia D'Angelo"}) {
		t.Errorf("person = %+v", got)
	}
	if got := employer(lines[0]); got != (payslip.Party{ID: vat, Name: "ACME ESEMPIO S.r.l."}) {
		t.Errorf("employer = %+v", got)
	}
}

// TestNoPersonWithoutCode leaves the person unknown without a codice
// fiscale, whatever names are printed, and reads an employer named under
// its label, with its code alone on a line.
func TestNoPersonWithoutCode(t *testing.T) {
	vat := synth.VATNumber("0123456789")
	pg := page(
		[]string{"40:Azienda"},
		[]string{"40:FONDAZIONE ESEMPIO ENTE DEL TERZO SETTORE"},
		[]string{"40:" + vat},
		[]string{"40:Dipendente", "100:BIANCHI ANNA"},
	)
	lines := [][]line{buildLines(pg)}
	if got := person(lines); got != (payslip.Party{}) {
		t.Errorf("person = %+v", got)
	}
	if got := employer(lines[0]); got != (payslip.Party{ID: vat, Name: "FONDAZIONE ESEMPIO ENTE DEL TERZO SETTORE"}) {
		t.Errorf("employer = %+v", got)
	}
}

func TestValidVAT(t *testing.T) {
	v := synth.VATNumber("0123456789")
	bad := v[:10] + string('0'+(v[10]-'0'+1)%10)
	for s, want := range map[string]bool{v: true, bad: false, "00000000000": false, "1234567890": false, v + "1": false} {
		if validVAT(s) != want {
			t.Errorf("validVAT(%q) = %v", s, !want)
		}
	}
}
