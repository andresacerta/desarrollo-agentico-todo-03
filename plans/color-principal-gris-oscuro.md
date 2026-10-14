# Feature: Cambiar el color principal de la web a un gris oscuro

## Encargo y criterios de aceptación

Issue: https://github.com/andresacerta/desarrollo-agentico-todo-03/issues/2 (#2) — "Cambiar el color principal de la web a un gris oscuro". El cuerpo está vacío.

Requisito explícito:

- El color principal de la web pasa de rojo teja (`#b73526`) a un gris oscuro.

Inferencias (el cuerpo no las detalla; se toman como decisiones razonables):

- "Color principal" es el token `--color-primary` de `public/assets/app.css` y todo lo que hoy actúa como acento de marca: punto de la marca `todo.`, botón primario, signo `+` de nueva tarea, enlace activo de la barra lateral y el hover del círculo de completar tarea.
- Los colores semánticos **no** cambian: peligro (`--color-danger`), prioridad alta (`--priority-high*`), error de campo, flash de éxito y foco (`--color-focus`, azul, necesario para accesibilidad).
- El gris elegido es cálido, coherente con los neutros existentes (`#282826`, `#686864`): `#2f2f2d` (hover `#1c1c1b`). Contraste con blanco ≈ 13:1, cumple WCAG AA/AAA.

Criterios de aceptación:

1. El botón "Añadir"/"Guardar" (`.button-primary`) tiene fondo gris oscuro y texto blanco; en hover es un gris aún más oscuro.
2. El punto de la marca y el `+` de nueva tarea se muestran en gris oscuro.
3. El enlace activo "Tareas" de la barra lateral usa un fondo gris claro neutro y texto gris oscuro, sin tintes rojizos.
4. El hover del círculo de completar tarea usa borde/icono gris oscuro y fondo gris claro neutro.
5. Los badges de prioridad alta, el botón de borrar en hover y los errores siguen en rojo.
6. El navegador carga la hoja nueva sin caché antigua (parámetro de versión actualizado).

## Comportamiento actual y propuesta

Actual: `:root` define `--color-primary: #b73526` y `--color-primary-hover: #992b1e`. Además hay tintes rojizos codificados a mano ligados al primario: `.nav-link` (`background: #f7e9e4`, `color: #a03123`, hover `#f2ded6`) y `.task-toggle-button:hover .completion-icon` (`background: #f7e9e4`). La plantilla carga `/assets/app.css?v=4`.

Propuesta: cambiar los dos tokens primarios a gris oscuro, sustituir los tintes rojizos por grises neutros (usando `var(--color-primary)` donde el valor es el primario) e incrementar la versión del CSS a `v=5`.

## Alcance

Incluido:

- Tokens `--color-primary` y `--color-primary-hover`.
- Tintes asociados al primario en `.nav-link`, `.nav-link:hover` y hover de `.completion-icon`.
- Versión del enlace a la hoja de estilos en `templates/application.plush.html`.

Excluido:

- Colores semánticos (peligro, prioridades, error, éxito, foco).
- Cambios de layout, tipografía, JavaScript, Go, modelos, migraciones o rutas.
- Tema oscuro o fondo de página oscuro (el encargo habla del color principal, no del fondo).

## Diseño por capas

CSS (`public/assets/app.css`), bloque `:root`:

```css
--color-primary: #2f2f2d;
--color-primary-hover: #1c1c1b;
```

Añadir un token para el fondo suave del primario, sustituyendo los tintes rojizos repetidos:

```css
--color-primary-soft: #ededea;
--color-primary-soft-hover: #e2e2de;
```

Reglas:

- `.nav-link`: `background: var(--color-primary-soft); color: var(--color-primary);`
- `.nav-link:hover`: `background: var(--color-primary-soft-hover);`
- `.task-toggle-button:hover .completion-icon`: `background: var(--color-primary-soft);` (borde y color ya usan `var(--color-primary)`).

`.brand-dot`, `.add-mark` y `.button-primary` ya usan las variables y no necesitan cambios.

Plantilla (`templates/application.plush.html`): `href="/assets/app.css?v=4"` → `?v=5`.

## Archivos relevantes

- `public/assets/app.css` — tokens de color y estilos de toda la interfaz.
- `templates/application.plush.html` — layout; enlaza la hoja con parámetro de versión para invalidar caché.
- `actions/tasks_test.go`, `actions/actions_test.go` — tests de las páginas; no comprueban colores, sirven de regresión de renderizado.

## Implementación paso a paso

1. En `public/assets/app.css`, dentro de `:root`, cambiar `--color-primary` a `#2f2f2d` y `--color-primary-hover` a `#1c1c1b`, y añadir justo debajo `--color-primary-soft: #ededea;` y `--color-primary-soft-hover: #e2e2de;`.
2. En `.nav-link`, sustituir `background: #f7e9e4;` por `background: var(--color-primary-soft);` y `color: #a03123;` por `color: var(--color-primary);`.
3. En `.nav-link:hover`, sustituir `background: #f2ded6;` por `background: var(--color-primary-soft-hover);`.
4. En `.task-toggle-button:hover .completion-icon`, sustituir `background: #f7e9e4;` por `background: var(--color-primary-soft);`.
5. Comprobar con `grep -nE '#f7e9e4|#f2ded6|#a03123|#b73526|#992b1e' public/assets/app.css` que solo queda `#b73526` en `--priority-high` (semántico, se mantiene).
6. En `templates/application.plush.html`, cambiar `/assets/app.css?v=4` por `/assets/app.css?v=5`.

## Tests y validación

No se añaden tests automáticos: el cambio es exclusivamente de valores CSS y el repositorio no tiene tests visuales ni de CSS; un test que buscara un hex en el CSS solo duplicaría la implementación.

Validaciones:

- `grep` del paso 5.
- `go build ./...` — la plantilla se embebe/compila con la app.
- `go test -p 1 ./...` — regresión de renderizado de las páginas (requiere PostgreSQL y migraciones de test, ver `database.yml`/`compose.yaml`). Si la base de datos no está disponible, informarlo.
- Verificación manual (si es posible levantar el entorno): abrir `/` y la edición de una tarea; comprobar los criterios 1–5 y que el foco con teclado sigue viéndose en azul.

## Riesgos y recuperación

- Caché del navegador mostrando el rojo antiguo: mitigado con `?v=5`.
- Pérdida de distinción entre el texto (`#282826`) y el primario (`#2f2f2d`) en la marca: el punto deja de destacar; es consecuencia esperada del encargo. Si se quisiera más contraste, ajustar solo `--color-primary`.
- Recuperación: revertir el commit; no hay datos ni migraciones implicados.
