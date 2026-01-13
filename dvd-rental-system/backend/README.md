
# Sistema de Renta de DVD (Go + Kubernetes + Electron)

Este proyecto es un sistema completo de gestión de rentas de DVD. Consta de un **Backend** RESTful desarrollado en Go, una **Base de Datos** PostgreSQL pre-cargada y un **Frontend** de escritorio desarrollado con Electron. Todo el sistema backend se despliega sobre un clúster local de **Kubernetes** (Minikube).

##  Arquitectura

* **Backend:** API REST en Go (Golang) usando `gorilla/mux`.
* **Base de Datos:** PostgreSQL 16 (contenedorizado con datos de ejemplo `dvdrental`).
* **Frontend:** Aplicación de escritorio multiplataforma (Electron + HTML/JS).
* **Infraestructura:** Kubernetes (Minikube) con orquestación de Pods y Servicios.

---

##  Requisitos Previos

Asegúrate de tener instaladas las siguientes herramientas:

1.  **Docker** (Motor de contenedores).
2.  **Minikube** (Clúster local de Kubernetes).
3.  **kubectl** (Línea de comandos para Kubernetes).
4.  **Node.js & npm** (Para ejecutar el Frontend).

---

##  Guía de Despliegue (Paso a Paso)

### 1. Iniciar el Entorno
Inicia tu clúster local:

minikube start

```

### 2. Preparar las Imágenes Docker (Solo la primera vez)

Para que Kubernetes pueda ver tus imágenes sin subirlas a Internet, configuramos el entorno de Docker para que apunte a Minikube:

eval $(minikube docker-env)

```

**A) Construir imagen de Base de Datos (con datos incluidos):**
Asegúrate de estar en la carpeta `backend` y tener el archivo `backend/db-init/dvdrental.tar`.

*(Nota: Si no tienes un `Dockerfile.db` específico, crea uno temporal con: `FROM postgres:16 \n COPY db-init/ /docker-entrypoint-initdb.d/`)*
cd backend
docker build -t my-dvd-db:v1 -f Dockerfile.db .

```

**B) Construir imagen del Backend:**
docker build -t my-dvd-api:v1 .

```

### 3. Desplegar en Kubernetes

Aplica los manifiestos de configuración (ubicados en la carpeta `k8s`):
kubectl apply -f k8s/database.yaml
kubectl apply -f k8s/backend.yaml

```

Verifica que los pods estén corriendo (STATUS: Running):
kubectl get pods

```

---

## Ejecución del Frontend (Escritorio)

### 1. Obtener la URL del Backend

Como Minikube asigna una IP dinámica, necesitas saber dónde está escuchando tu API:

minikube service api-service --url

```

*Copia la dirección que aparezca (ejemplo: `http://192.168.49.2:30123`).*

### 2. Configurar el Frontend

Abre el archivo `frontend-electron/index.html` y actualiza la constante `API` con la URL que copiaste:

```javascript
const API = '[http://192.168.49.2:30123](http://192.168.49.2:30123)'; // Tu URL de Minikube

```

### 3. Iniciar la Aplicación

Desde una nueva terminal, navega a la carpeta del frontend e inicia la app:

```
cd frontend-electron
npm install  # Solo la primera vez
npm start

```

---

## Endpoints de la API

La API expone los siguientes recursos principales:

### Autenticación

* `POST /api/auth/login` - Login (Roles: "staff", "customer").

### Rentas

* `POST /api/rentals` - Crear nueva renta.
* `POST /api/returns/{rental_id}` - Devolver una renta.
* `POST /api/rentals/{rental_id}/cancel` - Cancelar renta (si no se ha devuelto).

### Inventario

* `GET /api/inventory/available?film_id=ID` - Buscar copias disponibles.

### Reportes

* `GET /api/reports/customer/{id}/rentals` - Historial de cliente.
* `GET /api/reports/not-returned` - DVDs pendientes de devolución.
* `GET /api/reports/top-rented?limit=10` - Películas más rentadas.
* `GET /api/reports/revenue-by-staff` - Ventas por empleado.

---

## Comandos Útiles

**Ver logs del backend:**
kubectl logs -l app=dvd-api

```

**Detener el entorno:**
minikube stop

```

**Eliminar todo el clúster:**

minikube delete

