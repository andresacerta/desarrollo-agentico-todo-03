package models

import "strings"

func (ms *ModelSuite) Test_Task_Create_Valid() {
	task := &Task{Title: "Comprar café"}

	verrs, err := ms.DB.ValidateAndCreate(task)
	ms.NoError(err)
	ms.False(verrs.HasAny())

	ms.NotZero(task.ID)
	ms.False(task.Completed)

	count, err := ms.DB.Count(&Task{})
	ms.NoError(err)
	ms.Equal(1, count)
}

func (ms *ModelSuite) Test_Task_Title_Required() {
	task := &Task{Title: ""}

	verrs, err := ms.DB.ValidateAndCreate(task)
	ms.NoError(err)
	ms.True(verrs.HasAny())
	ms.Contains(verrs.Get("title"), "El título es obligatorio.")

	count, err := ms.DB.Count(&Task{})
	ms.NoError(err)
	ms.Equal(0, count)
}

func (ms *ModelSuite) Test_Task_Title_OnlySpaces_Invalid() {
	task := &Task{Title: "   \t  "}

	verrs, err := ms.DB.ValidateAndCreate(task)
	ms.NoError(err)
	ms.True(verrs.HasAny())
	ms.Contains(verrs.Get("title"), "El título es obligatorio.")
}

func (ms *ModelSuite) Test_Task_Title_MaxLength() {
	task := &Task{Title: strings.Repeat("a", TitleMaxLength)}

	verrs, err := ms.DB.ValidateAndCreate(task)
	ms.NoError(err)
	ms.False(verrs.HasAny())
}

func (ms *ModelSuite) Test_Task_Title_TooLong_Invalid() {
	task := &Task{Title: strings.Repeat("a", TitleMaxLength+1)}

	verrs, err := ms.DB.ValidateAndCreate(task)
	ms.NoError(err)
	ms.True(verrs.HasAny())
	ms.NotEmpty(verrs.Get("title"))
}

func (ms *ModelSuite) Test_Task_Toggle_Persists() {
	task := &Task{Title: "Revisar el correo"}
	verrs, err := ms.DB.ValidateAndCreate(task)
	ms.NoError(err)
	ms.False(verrs.HasAny())

	task.Completed = true
	ms.NoError(ms.DB.Update(task))

	reloaded := &Task{}
	ms.NoError(ms.DB.Find(reloaded, task.ID))
	ms.True(reloaded.Completed)
	ms.Equal(PriorityMedium, reloaded.Priority)
}

func (ms *ModelSuite) Test_Task_Priority_DefaultsToMedium() {
	task := &Task{Title: "Sin prioridad"}

	verrs, err := ms.DB.ValidateAndCreate(task)
	ms.NoError(err)
	ms.False(verrs.HasAny())

	ms.NoError(ms.DB.Reload(task))
	ms.Equal(PriorityMedium, task.Priority)
}

func (ms *ModelSuite) Test_Task_Priority_AcceptsAllLevels() {
	for _, priority := range []string{PriorityHigh, PriorityMedium, PriorityLow} {
		task := &Task{Title: "Tarea " + priority, Priority: priority}

		verrs, err := ms.DB.ValidateAndCreate(task)
		ms.NoError(err)
		ms.False(verrs.HasAny(), priority)

		ms.NoError(ms.DB.Reload(task))
		ms.Equal(priority, task.Priority)
	}
}

func (ms *ModelSuite) Test_Task_Priority_Invalid() {
	task := &Task{Title: "Prioridad inventada", Priority: "urgent"}

	verrs, err := ms.DB.ValidateAndCreate(task)
	ms.NoError(err)
	ms.True(verrs.HasAny())
	ms.Contains(verrs.Get("priority"), "Elige una prioridad válida.")

	count, err := ms.DB.Count(&Task{})
	ms.NoError(err)
	ms.Equal(0, count)
}

func (ms *ModelSuite) Test_Task_Priority_DatabaseRejectsUnknownValue() {
	err := ms.DB.Create(&Task{Title: "Sin validar", Priority: "urgent"})
	ms.Error(err)
}

func (ms *ModelSuite) Test_Task_PriorityLabel() {
	ms.Equal("Alta", (&Task{Priority: PriorityHigh}).PriorityLabel())
	ms.Equal("Media", (&Task{Priority: PriorityMedium}).PriorityLabel())
	ms.Equal("Baja", (&Task{Priority: PriorityLow}).PriorityLabel())
}
