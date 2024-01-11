package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
)

type Diagnostics diag.Diagnostics

func (d *Diagnostics) Contains(other diag.Diagnostic) bool {
	return (*diag.Diagnostics)(d).Contains(other)
}

func (d *Diagnostics) Append(toAppend ...diag.Diagnostic) {
	(*diag.Diagnostics)(d).Append(toAppend...)
}

func (d *Diagnostics) AddError(summary, detail string) {
	d.Append(NewErrorDiagnostic(summary, detail))
}

func (d *Diagnostics) AddAttributeError(path path.Path, summary, detail string) {
	(*diag.Diagnostics)(d).AddAttributeError(path, summary, detail)
}

func (d *Diagnostics) AddWarning(summary, detail string) {
	d.Append(NewWarningDiagnostic(summary, detail))
}

func (d *Diagnostics) HasError() bool {
	for _, cur := range *d {
		if cur.Severity() == diag.SeverityError {
			return true
		}
	}

	return false
}

func (d *Diagnostics) Errors() Diagnostics {
	return Diagnostics((*diag.Diagnostics)(d).Errors())
}

func (d *Diagnostics) AppendErrorReturn(summary, detail string) Diagnostics {
	return d.AppendReturn(diag.NewErrorDiagnostic(summary, detail))
}

func (d *Diagnostics) AppendReturn(toAppend ...diag.Diagnostic) Diagnostics {
	d.Append(toAppend...)
	return *d
}

func (d *Diagnostics) AppendCheckError(toAppend ...diag.Diagnostic) bool {
	d.Append(toAppend...)
	return d.HasError()
}

func NewErrorDiagnostic(summary, detail string) diag.Diagnostic {
	return diag.NewErrorDiagnostic(summary, detail)
}

func NewWarningDiagnostic(summary, detail string) diag.Diagnostic {
	return diag.NewWarningDiagnostic(summary, detail)
}

func NewErrorDiagnostics(summary, detail string) Diagnostics {
	return Diagnostics{NewErrorDiagnostic(summary, detail)}
}

func NewWarningDiagnostics(summary, detail string) Diagnostics {
	return Diagnostics{NewWarningDiagnostic(summary, detail)}
}
