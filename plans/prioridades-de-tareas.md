# Feature: Prioridades en las tareas

## Encargo y criterios de aceptación

Encargo original: «Hay que meter prioridades a las tareas. Tiene que ser con un diseño moderno y user friendly. Presta atención a los tests para asegurar que todo funcione bien».

Requisitos explícitos:

- Las tareas tienen prioridad.
- La interfaz para elegirla y verla es moderna y fácil de usar.
- La funcionalidad queda protegida con tests suficientes.

Inferencias (decisiones de este plan, revisables antes de implementar):

- Tres niveles fijos: **Alta**, **Media** y **Baja**. Valor por defecto: **Media**. No hay nivel «sin prioridad».
- La prioridad se elige al crear y al editar una tarea, y se ve en cada fila de la lista.
- La lista se ordena por prioridad (Alta → Media → Baja) y, dentro de la misma prioridad, por el orden actual (`created_at asc, id asc`). El estado completado no cambia el orden (como hoy).

Criterios de aceptación:

1. El formulario de «Añadir tarea» muestra un selector segmentado accesible con Alta / Media / Baja; Media aparece marcada por defecto.
2. Crear una tarea guarda la prioridad elegida. Si la petición no envía prioridad, se guarda Media.
3. Un valor de prioridad no reconocido (p. ej. `urgent`) no crea la tarea: responde 422 con el mensaje «Elige una prioridad válida.».
4. Si la creación falla por el título, el formulario re-renderizado conserva la prioridad que el usuario había marcado.
5. Cada fila de la lista muestra una etiqueta de prioridad con color e icono, con texto accesible «Prioridad alta/media/baja».
6. La lista aparece ordenada Alta → Media → Baja y, a igual prioridad, por fecha de creación ascendente.
7. La pantalla de edición muestra el mismo selector con la prioridad actual marcada y permite cambiar título y prioridad a la vez.
8. Una petición de actualización sin campo `priority` conserva la prioridad existente (compatibilidad con clientes que solo envían el título). Un valor no reconocido responde 422 y no cambia nada.
9. Completar/reabrir y eliminar no alteran la prioridad.
10. Las tareas existentes reciben prioridad Media al migrar; la base de datos rechaza valores fuera de `high`, `medium`, `low`.

## Comportamiento actual y propuesta

Estado actual (verificado en `main`, commit `ff94db6`, árbol limpio):

- `tasks` tiene `id`, `title`, `completed`, timestamps (`migrations/20260901000000_create_tasks.up.fizz`). No existe prioridad.
- `TasksIndex` y la rama de error de `TasksCreate` ordenan con el literal duplicado `"created_at asc, id asc"` (`actions/tasks.go`).
- `TasksCreate` hace `c.Bind(task)` directamente sobre `models.Task` (los campos no editables llevan `form:"-"`).
- `TasksUpdate` enlaza un struct anónimo solo con `Title` para no tocar el resto de campos; hay un test (`Test_TasksUpdate_PersistsTitleOnly`) que envía campos extra y comprueba que se ignoran.
- `templates/tasks/index.plush.html` pinta cada fila como grid `44px | título | acciones`; `public/assets/app.js` añade/quita `.task-status` insertándolo antes de `.task-actions` y lee `.task-title`.
- La hoja de estilos usa tokens en `:root`, objetivos táctiles de 44px, breakpoints 760px/380px y `prefers-reduced-motion`.

Propuesta: cambio vertical mínimo siguiendo los mismos patrones.

- Columna `priority` (`string`, `high|medium|low`, default `medium`, `CHECK`) añadida con una migración Fizz nueva.
- Campo `Priority string` en `models.Task` con constantes, opciones para la vista, valor por defecto en `BeforeValidate`/`BeforeSave` y validación en `Validate`.
- Orden por prioridad mediante una única constante de orden en `actions`.
- Selector segmentado con radios nativos (funciona sin JavaScript, navegable con flechas) en creación y edición; etiqueta de prioridad en cada fila.

Se usa `string` en vez de un entero para que un valor no numérico en el formulario no haga fallar `c.Bind` con un 500: la validación del modelo lo convierte en un 422 legible. El coste es una expresión `CASE` en el `ORDER BY`, aceptable para el tamaño de la lista.

## Alcance

Incluido:

- Migración up/down de la columna con restricción `CHECK`.
- Modelo: campo, constantes, etiquetas, default, validación.
- Acciones: crear, editar (render), actualizar y listar con prioridad; orden de la lista.
- Plantillas de lista y edición; CSS del selector y de la etiqueta; bump de versión del CSS.
- Tests de modelo y de acciones descritos en «Tests y validación».

Excluido:

- Filtros por prioridad, cambio de prioridad en línea desde la lista, arrastrar para reordenar, fechas de vencimiento.
- Mover las tareas completadas al final.
- Cambios en `public/assets/app.js` (no son necesarios; ver riesgos).
- Modificar migraciones existentes ni el seed `20260901000001_seed_tasks`.
- Corregir el bug preexistente del toggle asíncrono (ver «Riesgos»).

## Diseño por capas

### Persistencia y migración

Nuevo par `migrations/20261014000000_add_priority_to_tasks.{up,down}.fizz`:

```fizz
add_column("tasks", "priority", "string", {"size": 10, "default": "medium"})
sql("ALTER TABLE tasks ADD CONSTRAINT tasks_priority_check CHECK (priority IN ('high', 'medium', 'low'))")
```

```fizz
sql("ALTER TABLE tasks DROP CONSTRAINT IF EXISTS tasks_priority_check")
drop_column("tasks", "priority")
```

Fizz crea la columna `NOT NULL` por defecto; con `default` PostgreSQL rellena las filas existentes con `medium`. El nombre del archivo debe ordenar después de `20260901000001_seed_tasks`. `migrations/embed.go` ya incluye `*.fizz`, así que `models.Migrate()` la aplicará al arrancar. No se añade índice: la lista es pequeña y siempre se lee entera.

### Modelo (`models/task.go`)

```go
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
```

- Campo: `Priority string \`json:"priority" db:"priority" form:"priority"\``.
- `func (t *Task) PriorityLabel() string`: devuelve la etiqueta de `PriorityOptions` (cadena vacía si no reconocida; nunca debería ocurrir con datos persistidos).
- `func (t *Task) BeforeValidate(tx *pop.Connection) error` y `BeforeSave(...)`: si `strings.TrimSpace(t.Priority) == ""`, asigna `PriorityMedium`. Así `ValidateAndCreate`, `Create` y `Update` nunca envían `""` a la base de datos (que violaría el `CHECK`), y los tests existentes que crean `Task{Title: …}` siguen funcionando.
- `Validate`: si `Priority` no está en `PriorityOptions`, `verrs.Add("priority", "Elige una prioridad válida.")`.

### Acciones (`actions/tasks.go`)

- Constante de orden reutilizada por `TasksIndex` y la rama 422 de `TasksCreate`:

  ```go
  // taskListOrder ordena por prioridad (alta primero) y después por antigüedad.
  const taskListOrder = "CASE priority WHEN 'high' THEN 0 WHEN 'medium' THEN 1 ELSE 2 END, created_at asc, id asc"
  ```

- `c.Set("priorities", models.PriorityOptions)` en `TasksIndex`, en la rama 422 de `TasksCreate`, en `TasksEdit` y en la rama 422 de `TasksUpdate`.
- `TasksIndex`: el `task` vacío del formulario se inicializa con `Priority: models.PriorityMedium` para que Media salga marcada.
- `TasksCreate`: tras `c.Bind(task)`, recortar espacios de `task.Priority`; el default lo aplica el modelo.
- `TasksUpdate`: añadir `Priority string \`form:"priority" json:"priority"\`` al struct anónimo. Si `strings.TrimSpace(input.Priority) != ""`, asignarlo a `task.Priority`; si viene vacío, conservar el actual. Actualizar el comentario de la función («actualiza el título y la prioridad, conservando los demás datos»).
- En los errores 422 los mensajes de `title` y `priority` se muestran por separado.
- `TasksToggle` y `TasksDestroy` no cambian.

No hay rutas nuevas.

### Renderizado (Plush)

`templates/tasks/index.plush.html`:

- En cada `<li class="task …">`, envolver título y etiqueta en un contenedor `.task-main` (flex con `wrap`) que ocupa la columna central del grid; `.task-title` conserva su clase (la usa `app.js`). La etiqueta:

  ```html
  <span class="priority-badge priority-<%= t.Priority %>">
    <svg aria-hidden="true" …><!-- icono de bandera --></svg>
    <span class="sr-only">Prioridad </span><%= t.PriorityLabel() %>
  </span>
  ```

  Para el texto accesible se usa «Prioridad Alta/Media/Baja» (los tests buscan la etiqueta visible y la clase).
- En el formulario de creación, debajo de `.new-task-row`, un `fieldset.priority-picker` con `legend` «Prioridad» (visible, discreta) y, por cada opción de `priorities`, un `<label class="priority-option priority-<value>">` con `<input type="radio" name="priority" value="<value>" <checked si task.Priority == value>>`, un punto de color y el texto.
- Mostrar `errors.Get("priority")` en un `<p id="priority-error" class="field-error" role="alert">` y, si hay error, `aria-describedby="priority-error"` en el `fieldset`. Cambiar la condición actual del error de título a `errors.Get("title")` para que el `aria-invalid` del input solo se active con errores de título (hoy usa `errors.HasAny()`).

`templates/tasks/edit.plush.html`:

- Añadir el mismo `fieldset.priority-picker` entre la ayuda/error del título y `.edit-task-actions`, con la prioridad actual marcada.
- Cambiar la ayuda a «Hasta 200 caracteres.» y mover «Los demás datos de la tarea no cambian.» a la ayuda del fieldset o eliminarla (el estado completado sigue sin cambiar). Mantener `aria-describedby="title-help"`.
- Ajustar la condición del error de título a `errors.Get("title")`, igual que en la lista.

Para no duplicar el marcado del selector, crear el parcial `templates/tasks/_priority_picker.plush.html` y usarlo en ambas vistas con `<%= partial("tasks/priority_picker.plush.html") %>` (mismo mecanismo que `partial("flash.plush.html")` en el layout; el parcial lee `task`, `priorities` y `errors` del contexto).

### CSS (`public/assets/app.css`)

Nuevos tokens en `:root` (contraste AA sobre su fondo):

```css
--priority-high: #b73526;      --priority-high-bg: #fbece8;
--priority-medium: #9a5b00;    --priority-medium-bg: #fdf3e1;
--priority-low: #4a6a8a;       --priority-low-bg: #edf2f7;
```

- `.task-main`: `display: flex; flex-wrap: wrap; align-items: center; gap: 0.25rem 0.6rem; min-width: 0;` (alinea con el `padding` actual de `.task-title`).
- `.priority-badge`: pastilla `inline-flex`, `font-size: 0.75rem`, `font-weight: 600`, `padding: 0.1rem 0.5rem`, `border-radius: 999px`, color/fondo según `.priority-high|medium|low`, icono de 12px.
- `.task-completed .priority-badge`: tono atenuado (`color: var(--color-muted); background: #f2f2ee`).
- `.priority-picker`: sin borde de fieldset, `display: flex; gap: 0.4rem; flex-wrap: wrap`, alineado con el input (margen izquierdo de 44px/36px igual que `.add-mark` en el formulario de creación; sin margen en edición).
- `.priority-option`: control segmentado tipo «chip»; `min-height: 44px` (objetivo táctil), borde `var(--color-border)`, `border-radius: 999px`, punto de color con `::before`. El `input` es `sr-only`-like pero enfocable (posición absoluta, opacidad 0, sin `display:none`).
- `.priority-option:has(input:checked)`: fondo y borde del color de la prioridad, texto en su color.
- `.priority-option:has(input:focus-visible)`: anillo `var(--color-focus)`, igual que `input:focus-visible`.
- Transición `background-color/border-color 120ms` (ya neutralizada por la regla `prefers-reduced-motion`).
- En `@media (max-width: 760px)` reducir el margen izquierdo del picker a 36px; en 380px permitir que los chips ocupen el ancho (`flex: 1`).

Actualizar `templates/application.plush.html`: `app.css?v=3` → `app.css?v=4`.

### JavaScript

Sin cambios. `app.js` sigue funcionando porque `.task-title` y `.task-actions` se conservan y `.task-status` se inserta antes de `.task-actions`, que sigue siendo hijo directo de `.task`.

## Archivos relevantes

| Ruta | Responsabilidad |
| --- | --- |
| `migrations/20261014000000_add_priority_to_tasks.up.fizz` (nuevo) | Añade la columna con default y `CHECK` |
| `migrations/20261014000000_add_priority_to_tasks.down.fizz` (nuevo) | Revierte la restricción y la columna |
| `models/task.go` | Campo, constantes, opciones, etiqueta, default y validación |
| `models/task_test.go` | Tests de modelo de prioridad |
| `actions/tasks.go` | Orden de la lista, binding de prioridad en crear/actualizar, opciones para la vista |
| `actions/tasks_test.go` | Tests HTTP de la funcionalidad |
| `fixtures/tasks.toml` | Escenario nuevo con prioridades mezcladas |
| `templates/tasks/_priority_picker.plush.html` (nuevo) | Selector segmentado reutilizable |
| `templates/tasks/index.plush.html` | Etiqueta en cada fila y selector en el alta |
| `templates/tasks/edit.plush.html` | Selector en la edición |
| `templates/application.plush.html` | Bump de versión del CSS |
| `public/assets/app.css` | Tokens, etiqueta, selector y responsive |
| `migrations/embed.go`, `models/models.go` | Sin cambios; confirman que la migración se embebe y se aplica al arrancar |

## Implementación paso a paso

1. Comprobar `git status --short --branch` y que `main` sigue en el estado descrito.
2. Crear las dos migraciones Fizz del apartado de persistencia.
3. Modelo: añadir constantes, `PriorityOption`, `PriorityOptions`, campo `Priority`, `PriorityLabel`, `BeforeValidate`, `BeforeSave` y la validación en `Validate`.
4. Tests de modelo (sección siguiente) y ejecutarlos.
5. Acciones: constante `taskListOrder` sustituyendo ambos literales de orden; `priorities` en el contexto de las cuatro vistas renderizadas; default Media en el formulario vacío; trim de prioridad al crear; prioridad opcional en `TasksUpdate`.
6. Crear `_priority_picker.plush.html` y usarlo en `index` y `edit`; añadir la etiqueta en las filas con el contenedor `.task-main`; separar los errores por campo.
7. CSS y bump `v=4` del layout.
8. Añadir el escenario de fixtures y los tests de acciones; actualizar los tests existentes indicados.
9. Ejecutar la validación completa y revisar visualmente en escritorio y a 375px (crear, editar, completar, error de título conservando prioridad, navegación por teclado del selector).
10. Revisar `git diff --check`, `git diff` y `git status --short`.

## Tests y validación

### Preparación

Los tests necesitan PostgreSQL (`compose.yaml`: `docker compose up -d db`) y la base `todo_test` migrada, como en CI:

```bash
go install github.com/gobuffalo/pop/v6/soda@v6.4.1
GO_ENV=test soda migrate -e test
```

Repetir `soda migrate -e test` tras crear la migración. Comprobar también la reversibilidad una vez: `GO_ENV=test soda migrate down -e test` seguido de `soda migrate -e test`.

### Tests de modelo (`models/task_test.go`)

- `Test_Task_Priority_DefaultsToMedium`: `Task{Title}` sin prioridad → se crea, `Priority == PriorityMedium` tras `Reload`.
- `Test_Task_Priority_AcceptsAllLevels`: tabla con `high`, `medium`, `low` → sin errores y persistido.
- `Test_Task_Priority_Invalid`: `Priority: "urgent"` → `verrs.Get("priority")` contiene «Elige una prioridad válida.» y `Count == 0`.
- `Test_Task_Priority_DatabaseRejectsUnknownValue`: `ms.DB.Create(&Task{Title: "x", Priority: "urgent"})` (sin validación) devuelve error → protege el `CHECK` de la migración.
- `Test_Task_PriorityLabel`: devuelve «Alta», «Media», «Baja».
- `Test_Task_Toggle_Persists` existente: añadir `ms.Equal(PriorityMedium, reloaded.Priority)`.

### Tests de acciones (`actions/tasks_test.go`)

Escenario nuevo en `fixtures/tasks.toml`, `"prioridades mezcladas"`, con `created_at` explícitos y crecientes (p. ej. `2026-09-01 09:00:00`, `09:01`, `09:02`, `09:03`) y prioridades `low`, `high`, `medium`, `high` en ese orden de creación, con títulos distinguibles.

- `Test_TasksIndex_OrdersByPriority`: cargar el escenario; con `strings.Index` comprobar el orden: high (09:01) < high (09:03) < medium < low. Cubre orden principal y desempate.
- `Test_TasksIndex_ShowsPriorityBadges`: el cuerpo contiene `priority-badge priority-high`, «Alta», «Media», «Baja».
- `Test_TasksIndex_NewTaskFormDefaultsToMedium`: el radio `value="medium"` aparece con `checked` y los de `high`/`low` no (comparar los fragmentos del input renderizado).
- `Test_TasksCreate_WithPriority`: POST `url.Values{"title": {"Urgente"}, "priority": {"high"}}` → 303 y `Priority == "high"`.
- `Test_TasksCreate_WithoutPriority_DefaultsToMedium`: POST solo con título → `Priority == "medium"`.
- `Test_TasksCreate_InvalidPriority`: `priority=urgent` → 422, «Elige una prioridad válida.», `Count == 0`.
- `Test_TasksCreate_EmptyTitle_KeepsSelectedPriority`: `title=""`, `priority=low` → 422 y el radio `low` aparece marcado.
- `Test_TasksEdit_PrefillsPriority`: tarea `high` → el radio `high` aparece `checked`.
- `Test_TasksUpdate_ChangesPriority`: PUT con `title` y `priority=low` → 303, ambos persistidos, `Completed` intacto.
- `Test_TasksUpdate_InvalidPriority`: PUT con `priority=urgent` → 422, mensaje visible, título y prioridad sin cambios.
- `Test_TasksToggle_KeepsPriority`: tarea `high`, PUT toggle → sigue `high`.
- Actualizar `Test_TasksUpdate_PersistsTitleOnly`: crear la tarea con `Priority: high` en `taskForEditing` (o en el propio test), enviar la petición sin `priority` como ahora y comprobar que sigue `high`. Renombrar a `Test_TasksUpdate_WithoutPriority_KeepsOtherFields` solo si el implementador lo considera necesario; el comportamiento verificado es el mismo más la prioridad.
- `Test_TasksUpdate_EmptyTitle` sigue exigiendo `aria-describedby="title-error"` en el input de título; debe seguir pasando tras separar errores por campo.

Comprobar que los tests actuales pasan sin otras modificaciones (`Test_TasksIndex_ListsTasks` usa el escenario «tres tareas», cuyas filas reciben el default `medium` de la base de datos).

### Validación

```bash
gofmt -l .
```

```bash
go vet ./...
```

```bash
go test -p 1 ./...
```

```bash
go build ./...
```

`gofmt -l .` no debe listar archivos. Revisión manual en el navegador con `docker compose up` o la skill `env:start`: el selector es operable con Tab + flechas, el foco es visible, y a 375px no hay desbordamiento horizontal.

## Riesgos y recuperación

- **Bug preexistente en el toggle asíncrono** (fuera de alcance): `public/assets/app.js` termina con `fetch(form.action)`, que hace un `GET` a `/tasks/{id}/toggle`, ruta que solo existe como `PUT`; el cambio visual no se persiste. No afecta a la prioridad, pero al probar manualmente parecerá que «completar» se pierde al recargar. Conviene corregirlo en un encargo aparte (`/bug`).
- **Restricción `CHECK` y escrituras sin default**: cualquier inserción que envíe `priority = ''` fallaría. Mitigado por `BeforeValidate`/`BeforeSave` y cubierto por tests. Si apareciera otra vía de escritura, debe pasar por el modelo.
- **`:has()` en CSS**: soportado por navegadores actuales. Sin él, el chip marcado no se resalta pero el radio sigue funcionando y siendo accesible; es una degradación aceptable.
- **Orden con `CASE` en SQL**: depende de que los valores sean exactamente los tres definidos; el `CHECK` lo garantiza.
- **Despliegue**: la migración es aditiva y rápida; el binario la aplica al arrancar (`models.Migrate()`). Rollback: `soda migrate down` (elimina la restricción y la columna, perdiendo las prioridades asignadas) y desplegar el binario anterior. Un binario anterior funciona también con la columna presente, porque la columna tiene default y no se selecciona.
- **Caché del navegador**: sin el bump `app.css?v=4` el nuevo marcado se vería sin estilos.
