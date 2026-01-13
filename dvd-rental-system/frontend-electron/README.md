# DVD Rental Frontend (Electron)

Aplicación de escritorio desarrollada con **Electron** (HTML, CSS, JS) que actúa como cliente para el sistema de renta de DVDs.

Se conecta a la API Backend alojada en Kubernetes (Minikube) para gestionar rentas, devoluciones e inventario.

##  Requisitos Previos

* **Node.js** (v18 o superior).
* **npm** (Viene instalado con Node.js).
* El Backend debe estar corriendo en Minikube.

##  Configuración de Conexión

Como Minikube asigna direcciones IP dinámicas, es necesario configurar la URL de la API antes de iniciar:

1.  **Obtener la URL de la API:**
    En tu terminal, con Minikube encendido, ejecuta:
    ```bash
    minikube service api-service --url
    ```
    *(Copia la dirección resultante, por ejemplo: `http://192.168.49.2:30123`)*.

2.  **Actualizar el código:**
    Abre el archivo `index.html` en esta carpeta.
    Busca la línea:
    ```javascript
    const API = '...';
    ```
    Y pega la URL que obtuviste en el paso anterior.

## Instalación y Ejecución

1.  **Instalar dependencias:**
    (Solo necesario la primera vez)
    ```bash
    npm install
    ```

2.  **Iniciar la aplicación:**
    ```bash
    npm start
    ```

##  Crear Acceso Directo (Ubuntu/Linux)

Para ejecutar la aplicación desde el escritorio sin abrir la terminal:

1.  Crea un archivo llamado `dvd-rental.desktop` en tu escritorio.
2.  Pega el siguiente contenido (ajustando la ruta a tu carpeta):

    ```ini
    [Desktop Entry]
    Version=1.0
    Name=DVD Rental System
    Comment=Sistema de Renta de DVDs
    Exec=bash -c "cd /ruta/a/tu/proyecto/frontend-electron && npm start"
    Icon=utilities-terminal
    Terminal=true
    Type=Application
    Categories=Utility;Application;
    ```
3.  Haz clic derecho en el archivo -> **Propiedades** -> **Permisos** -> Marca **"Permitir ejecutar como un programa"**.

---
**Nota:** Asegúrate de haber iniciado Minikube (`minikube start`) antes de abrir la aplicación.