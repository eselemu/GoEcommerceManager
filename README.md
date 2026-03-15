# Go Ecommerce Manager

This is a TCP-based Inventory/Ecommerce Management system written in Go. It consists of a Server and a Client application that communicate asynchronously using JSON over TCP.

---

## 🚀 Requisitos previos (Prerequisites)

Para correr el proyecto, asegúrese de tener instalada la última versión de **Go (Golang)** en su sistema.

De acuerdo a las instrucciones del administrador del proyecto, el único paquete externo que deben instalar manualmente es el siguiente comando. Ejecútelo en su terminal antes de iniciar:

```bash
go get github.com/google/uuid
```

*(Opcional: Si al intentar correr el sistema indica que le faltan dependencias como `bcrypt` listadas en el archivo `go.mod`, ejecute el comando `go mod tidy` para actualizar el repositorio local).*

---

## 🏃 Instrucciones para correr la aplicación (How to Run)

La aplicación funciona bajo un modelo Cliente-Servidor. Es necesario levantar **primero el Servidor** y posteriormente iniciar uno o más **Clientes** en paralelo.

### 1. Iniciar el Servidor (Start the Server)

1. Abra una terminal (consola) de comandos o PowerShell.
2. Navegue hasta la ruta raíz del proyecto (`GoEcommerceManager`).
3. Ejecute el siguiente comando para levantar el servidor:

```bash
go run Server.go
```

El servidor leerá la configuración de `config.json` y empezará a escuchar en el puerto indicado (ej. port 6969). Además cargará la base de datos desde los archivos JSON (`users.json`, `products.json`, `orders.json`).
**Importante:** No cierre esta terminal. El servidor debe mantenerse corriendo.

### 2. Iniciar el Cliente (Start the Client)

1. Sin cerrar el servidor, abra una **nueva ventana/pestaña** de terminal (consola).
2. Navegue nuevamente hasta la raíz del proyecto.
3. Ejecute el siguiente comando para iniciar el cliente interactivo:

```bash
go run Client.go
```

### 3. Iniciar Sesión (Login credentials)

Una vez iniciado el cliente, la consola le solicitará ingresar un usuario y contraseña. El servidor crea automáticamente dos perfiles por defecto y con diferentes permisos (Administrador y Cliente regular):

**Para acceder con cuenta de Administrador (Admin):**
- **Usuario (Username):** `Admin1`
- **Contraseña (Password):** `12345`

*Funciones de administrador: Los administradores pueden agregar (`add_product`), eliminar (`delete_product`), actualizar el inventario (`update_stock`), precios (`update_price`) y cambiar el estado de los pedidos (`update_order_status`).*

**Para acceder con cuenta de Cliente Regular (Customer):**
- **Usuario (Username):** `Client`
- **Contraseña (Password):** `12345`

*Funciones de cliente: Los clientes estándar pueden consultar y visualizar productos y generar pedidos (usando `add_to_cart` y `checkout`).*

### 🛑 Notas y Tips Adicionales
- Puede iniciar múltiples clientes abriendo terminales usando `go run Client.go` simultáneamente, todos se conectarán al mismo servidor.
- Asegúrese de que el puerto establecido en `config.json` en este caso (localhost:6969) no esté siendo utilizado por ningún otro programa en su PC.
- Para salir del ambiente del cliente limpiamente, simplemente escriba el comando y presione enter en su terminal de cliente:
```bash
exit
```
