# Pirateca API (qumran-api)

API en Go para Pirateca, un catálogo de libros en PDF/EPUB. Sirve `api.pirateca.com` y `api.pirateca.net`, y respalda al frontend en Next.js (`pirateca-fe`).

## Tabla de contenidos

- [Requisitos del sistema](#requisitos-del-sistema)
- [Dependencias externas (binarios)](#dependencias-externas-binarios)
- [Configuración (`config.yaml`)](#configuración-configyaml)
- [Compilación](#compilación)
- [Estructura de carpetas de uploads](#estructura-de-carpetas-de-uploads)
- [Migraciones de base de datos](#migraciones-de-base-de-datos)
- [Flags de arranque](#flags-de-arranque)
- [Servicio systemd](#servicio-systemd)
- [nginx (reverse proxy)](#nginx-reverse-proxy)
- [Checklist de verificación post-deploy](#checklist-de-verificación-post-deploy)
- [Problemas comunes y diagnóstico](#problemas-comunes-y-diagnóstico)

---

## Requisitos del sistema

- Go 1.22+ (definido en `go.mod`)
- PostgreSQL
- nginx como reverse proxy con HTTPS (certbot)
- systemd para correr el binario como servicio

## Dependencias externas (binarios)

El código invoca tres binarios externos vía `exec.Command` (ver `cmd/api/helpers.go`). **Si falta cualquiera de estos, la subida de un libro responde `500 Internal Server Error`** y el detalle exacto aparece en los logs de systemd (`executable file not found in $PATH`).

| Binario | Para qué se usa | Paquete Debian/Ubuntu |
|---|---|---|
| `exiftool` | Limpiar y reescribir metadatos de PDFs e imágenes | `libimage-exiftool-perl` |
| `transmission-create` | Generar el archivo `.torrent` del PDF | `transmission-cli` |
| `convert` (ImageMagick) | Convertir portadas a `.jpg` si no vienen en ese formato | `imagemagick` |

### Instalación en Debian / Ubuntu

```bash
sudo apt update
sudo apt install -y libimage-exiftool-perl transmission-cli imagemagick
```

### Instalación en Arch Linux

```bash
sudo pacman -Syu perl-image-exiftool transmission-cli imagemagick
```

> En Arch, `transmission-cli` provee `transmission-create` igual que en Debian. Si el paquete cambia de nombre en el futuro, busca con `pacman -Ss transmission`.

### Verificación (en ambos sistemas)

```bash
which exiftool transmission-create convert
```

Los tres deben aparecer con su ruta completa (normalmente bajo `/usr/bin/`).

### ⚠️ Nota sobre el `$PATH` de systemd

Aunque los binarios estén instalados y disponibles en la shell de tu usuario, **systemd puede arrancar el servicio con un `$PATH` distinto o más restringido**. Por eso el archivo de servicio (ver más abajo) debe fijar explícitamente:

```ini
Environment=PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin
```

Si en el futuro aparece un error de `executable file not found in $PATH` para un binario que `which` sí encuentra desde la terminal, esta es la causa más probable.

## Configuración (`config.yaml`)

El binario requiere un archivo `config.yaml` en el **working directory** desde el que se ejecuta (se lee con Viper, `viper.SetConfigFile("config.yaml")`). Si no existe, el programa hace `panic` al arrancar.

Esto es lo mínimo necesario — el campo `database.dsn` es el único que el código realmente usa para arrancar (los campos `smtp.*` son leídos por Viper pero solo se usan si en algún momento se activa el envío de correos; si no los configuras, simplemente quedan vacíos sin causar error):

```yaml
database:
  dsn: "postgres://USUARIO:CONTRASEÑA@localhost/NOMBRE_DB?sslmode=disable"
```

Ejemplo real de producción (con la contraseña reemplazada):

```yaml
database:
  dsn: "postgres://pirateca:TU_PASSWORD_AQUI@localhost/pirateca?sslmode=disable"
```

Si en algún momento se necesita envío de correo (recuperación de contraseña, notificaciones, etc.), se puede agregar:

```yaml
database:
  dsn: "postgres://pirateca:TU_PASSWORD_AQUI@localhost/pirateca?sslmode=disable"

smtp:
  host: "smtp.ejemplo.com"
  port: 587
  username: "usuario"
  password: "contraseña"
  sender: "Pirateca <no-reply@pirateca.com>"
```

Este archivo **no debe subirse a git** (ya está cubierto por `.gitignore` si sigue la convención del proyecto).

## Compilación

```bash
cd ~/qumran-api
go mod tidy
go build -ldflags='-s -w' -o ./bin/api ./cmd/api
```

El flag `-ldflags='-s -w'` reduce el tamaño del binario quitando símbolos de debug; es opcional pero se ha usado en deploys anteriores.

## Estructura de carpetas de uploads

**Importante:** todas las rutas de almacenamiento en el código son relativas (`./uploads/pdfs`, `./uploads/covers`, `./uploads/torrents`, `./uploads/torrentadded`, `./uploads/epubs`). Esto significa que:

- El proceso las crea automáticamente (`os.MkdirAll`) si no existen, **relativas al directorio desde el que se ejecuta el binario**.
- El `WorkingDirectory` del servicio systemd **debe ser** el directorio raíz del proyecto (ej. `/home/jesarx/qumran-api`), no el directorio `bin/`. Si esto está mal configurado, los archivos se crearán en un lugar inesperado o el proceso fallará al no tener permisos de escritura.

Estructura esperada en producción:

```
qumran-api/
├── bin/api
├── config.yaml
└── uploads/
    ├── pdfs/
    ├── covers/
    ├── epubs/
    ├── torrents/
    └── torrentadded/
```

Asegúrate de que el usuario que corre el servicio (`jesarx`) tenga permisos de escritura sobre `uploads/` y sus subcarpetas.

## Migraciones de base de datos

Las migraciones SQL están en `/migrations` (numeradas, con pares `.up.sql`/`.down.sql`). Si usas `golang-migrate`:

```bash
migrate -path ./migrations -database "postgres://pirateca:TU_PASSWORD@localhost/pirateca?sslmode=disable" up
```

## Flags de arranque

El binario acepta los siguientes flags (todos opcionales salvo que se necesite sobreescribir el default):

| Flag | Default | Descripción |
|---|---|---|
| `-port` | `4000` | Puerto del servidor HTTP |
| `-env` | `development` | `development`, `staging` o `production` |
| `-db-dsn` | (desde `config.yaml`) | DSN de PostgreSQL |
| `-db-max-open-conns` | `25` | Conexiones máximas abiertas |
| `-db-max-iddle-conns` | `25` | Conexiones máximas inactivas |
| `-db-max-iddle-time` | `1m` | Tiempo máximo de inactividad por conexión |
| `-limiter-rps` | `8` | Rate limit: requests por segundo |
| `-limiter-burst` | `16` | Rate limit: burst máximo |
| `-limiter-enabled` | `true` | Habilita/deshabilita el rate limiter |
| `-smtp-host`, `-smtp-port`, `-smtp-username`, `-smtp-password`, `-smtp-sender` | (desde `config.yaml`) | Configuración de envío de correo |
| `-cors-trusted-origins` | (vacío) | Orígenes permitidos para CORS, separados por espacio, entre comillas |

**Para producción, el comando mínimo necesario es:**

```bash
./bin/api \
  -env=production \
  -port=4000 \
  -cors-trusted-origins="https://pirateca.com https://pirateca.net"
```

Sin `-env=production`, el healthcheck (`/v1/healthcheck`) reportará `"enviroment":"development"` aunque todo lo demás funcione — es la señal más rápida de que el flag falta.

Sin `-cors-trusted-origins` apuntando exactamente a los dominios del frontend, el navegador bloqueará las peticiones por CORS (asegúrate de incluir `https://` y de no dejar `/` al final de cada origen).

## Servicio systemd

Archivo: `/etc/systemd/system/pirateca.service`

```ini
[Unit]
Description=Pirateca Go API
After=network.target postgresql.service

[Service]
Type=simple
User=jesarx
WorkingDirectory=/home/jesarx/qumran-api
ExecStart=/home/jesarx/qumran-api/bin/api \
  -env=production \
  -port=4000 \
  -cors-trusted-origins="https://pirateca.com https://pirateca.net"
Restart=on-failure
Environment=NODE_ENV=production
Environment=PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin

[Install]
WantedBy=multi-user.target
```

Puntos clave de este archivo:

- **`WorkingDirectory`** debe ser la raíz del proyecto, no `bin/` (ver sección de uploads arriba).
- **`Environment=PATH=...`** es necesario para que `exec.Command` encuentre `exiftool`, `transmission-create` y `convert` — sin esta línea, systemd puede usar un PATH mínimo que no los incluya aunque estén instalados.
- `Restart=on-failure` reinicia el proceso si crashea; considera `Restart=always` con `RestartSec`, `StartLimitIntervalSec` y `StartLimitBurst` si el VPS tiene historial de caídas por OOM.

Aplicar cambios:

```bash
sudo systemctl daemon-reload
sudo systemctl enable pirateca
sudo systemctl restart pirateca
sudo systemctl status pirateca
```

## nginx (reverse proxy)

Archivo típico: `/etc/nginx/sites-available/<nombre-del-sitio>` (verifica el nombre real con `ls /etc/nginx/sites-available/`, ya que ha variado entre reconstrucciones del VPS).

```nginx
server {
    listen 443 ssl;
    server_name api.pirateca.com api.pirateca.net;

    client_max_body_size 200M;

    location / {
        proxy_pass http://localhost:4000;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

**`client_max_body_size`** es crítico: sin esta línea (o con un valor muy bajo), nginx responde `413 Request Entity Too Large` al subir PDFs grandes, lo que en el navegador puede mostrarse como un `NetworkError` genérico en vez de un error HTTP claro.

Tras editar:

```bash
sudo nginx -t
sudo systemctl reload nginx
```

## Checklist de verificación post-deploy

Ejecutar en orden tras cualquier reconstrucción del VPS o redeploy:

1. **Binarios externos instalados** (ver comandos de instalación para [Debian/Ubuntu](#instalación-en-debian--ubuntu) o [Arch](#instalación-en-arch-linux) más arriba):
   ```bash
   which exiftool transmission-create convert
   ```
2. **`config.yaml` presente** en el working directory del servicio, con DSN correcto.
3. **Carpetas de `uploads/`** existen (o el usuario del servicio tiene permiso para crearlas) dentro del working directory correcto.
4. **Servicio corriendo en producción:**
   ```bash
   curl http://localhost:4000/v1/healthcheck
   ```
   Debe responder `"enviroment":"production"`.
5. **Accesible vía HTTPS y con headers CORS correctos:**
   ```bash
   curl -i https://api.pirateca.net/v1/healthcheck
   ```
6. **nginx permite el tamaño de subida esperado** (`client_max_body_size` ≥ tamaño máximo de PDF que subirás).
7. **Prueba real de subida** desde `https://pirateca.net/dashboard/books/new` con un PDF y una portada que no sea `.jpg` (para forzar el path de `convert`), confirmando que no hay errores en:
   ```bash
   sudo journalctl -u pirateca -f
   ```

## Problemas comunes y diagnóstico

| Síntoma en el navegador | Causa probable | Diagnóstico |
|---|---|---|
| `NetworkError when attempting to fetch resource` al subir un libro | nginx rechazó la petición por tamaño (`413`) antes de llegar a la API | `sudo tail -f /var/log/nginx/error.log` mientras subes el archivo; revisa `client_max_body_size` |
| `500 Internal Server Error` en `POST /v1/books` | Falta un binario externo (`exiftool`, `transmission-create`, `convert`) o el `$PATH` de systemd no lo incluye | `sudo journalctl -u pirateca -n 20 --no-pager` — el mensaje indica exactamente qué binario falta |
| Healthcheck muestra `"enviroment":"development"` en producción | Falta el flag `-env=production` en `ExecStart` | Revisar `/etc/systemd/system/pirateca.service` |
| CORS error en la consola del navegador | El origen del frontend no está en `-cors-trusted-origins`, o tiene un typo (slash final, falta `https://`) | Revisar el `ExecStart` del servicio |
| `pirateca-api.service` o `pirateca.service` "could not be found" | El nombre del servicio cambió entre reconstrucciones del VPS | `systemctl list-units --type=service --all \| grep -i pirateca` |
| Servicio en `failed (Result: exit-code)` con "Start request repeated too quickly" | Crash inmediato al arrancar (config faltante, DB inaccesible, binario mal compilado) | `cd ~/qumran-api && ./bin/api` (ejecutar manualmente para ver el error real sin que systemd lo oculte) |

---

*Última actualización: documentado tras resolver una cadena de fallos en cascada (nginx `413` → `exiftool` faltante → `transmission-create` faltante → `convert` faltante) en una subida de libro en producción, junio 2026.*
