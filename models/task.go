package models

import (
	"fmt"
	"strings"
	"time"

	"github.com/gobuffalo/pop/v6"
	"github.com/gobuffalo/validate/v3"
	"github.com/gofrs/uuid"
)

// TitleMaxLength es el número máximo de caracteres permitido en el título.
const TitleMaxLength = 200

// Prioridades admitidas para una tarea.
const (
	PriorityHigh   = "high"
	PriorityMedium = "medium"
	PriorityLow    = "low"
)

// PriorityOption describe una prioridad tal y como se ofrece en los formularios.
type PriorityOption struct {
	Value string
	Label string
}

// PriorityOptions enumera las prioridades en el orden en que se muestran.
var PriorityOptions = []PriorityOption{
	{PriorityHigh, "Alta"},
	{PriorityMedium, "Media"},
	{PriorityLow, "Baja"},
}

// Task representa una tarea de la lista.
type Task struct {
	ID        uuid.UUID `json:"id" db:"id" form:"-"`
	Title     string    `json:"title" db:"title" form:"title"`
	Completed bool      `json:"completed" db:"completed" form:"-"`
	Priority  string    `json:"priority" db:"priority" form:"priority"`
	CreatedAt time.Time `json:"created_at" db:"created_at" form:"-"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at" form:"-"`
}

// Tasks es una colección de Task.
type Tasks []Task

// PriorityLabel devuelve el nombre visible de la prioridad de la tarea.
func (t *Task) PriorityLabel() string {
	for _, option := range PriorityOptions {
		if option.Value == t.Priority {
			return option.Label
		}
	}
	return ""
}

// BeforeValidate asigna la prioridad por defecto antes de validar.
func (t *Task) BeforeValidate(tx *pop.Connection) error {
	t.applyDefaultPriority()
	return nil
}

// BeforeSave evita enviar una prioridad vacía cuando se guarda sin validar.
func (t *Task) BeforeSave(tx *pop.Connection) error {
	t.applyDefaultPriority()
	return nil
}

func (t *Task) applyDefaultPriority() {
	if strings.TrimSpace(t.Priority) == "" {
		t.Priority = PriorityMedium
	}
}

func isValidPriority(priority string) bool {
	for _, option := range PriorityOptions {
		if option.Value == priority {
			return true
		}
	}
	return false
}

// Validate se ejecuta en cada llamada a los métodos "pop.Validate*".
func (t *Task) Validate(tx *pop.Connection) (*validate.Errors, error) {
	verrs := validate.NewErrors()

	title := strings.TrimSpace(t.Title)
	if title == "" {
		verrs.Add("title", "El título es obligatorio.")
	} else if len([]rune(title)) > TitleMaxLength {
		verrs.Add("title", fmt.Sprintf("El título no puede superar los %d caracteres.", TitleMaxLength))
	}

	if !isValidPriority(t.Priority) {
		verrs.Add("priority", "Elige una prioridad válida.")
	}

	return verrs, nil
}
