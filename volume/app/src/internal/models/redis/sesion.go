package redis

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// SesionKeyPrefix es el prefijo base utilizado para las claves de sesión en Redis.
const SesionKeyPrefix = "sesion"

// Genero representa el género de la persona.
type Genero string

// AplicacionSlug representa el identificador único de una aplicación en el ecosistema.
type AplicacionSlug string

// BuildSesionKey genera la clave estandarizada de Redis con el formato sesion:usuario_id:timestamp.
func BuildSesionKey(usuarioID uuid.UUID, timestamp int64) string {
	return fmt.Sprintf("%s:%s:%d", SesionKeyPrefix, usuarioID, timestamp)
}

// Sesion representa la estructura de datos guardada en caché para una sesión activa en Redis.
type Sesion struct {
	Persona      SesionPersona      `json:"persona"`
	Miembro      *SesionMiembro     `json:"miembro,omitempty"`
	Usuario      SesionUsuario      `json:"usuario"`
	Aplicaciones []SesionAplicacion `json:"aplicaciones"`
}

// BuildKey retorna la clave de Redis para la sesión actual dado un timestamp.
func (s *Sesion) BuildKey(timestamp int64) string {
	return BuildSesionKey(s.Usuario.ID, timestamp)
}

// SesionComuna contiene la información resumida de la comuna de la persona en la sesión.
type SesionComuna struct {
	ID          uuid.UUID `json:"id"`
	Nombre      string    `json:"nombre"`
	ProvinciaID uuid.UUID `json:"provincia_id"`
}

// SesionPersona contiene los datos personales asociados a la sesión activa.
type SesionPersona struct {
	ID                uuid.UUID     `json:"id"`
	Run               *string       `json:"run,omitempty"`
	Pasaporte         *string       `json:"pasaporte,omitempty"`
	Nombre            string        `json:"nombre"`
	ApellidoPaterno   *string       `json:"apellido_paterno,omitempty"`
	ApellidoMaterno   *string       `json:"apellido_materno,omitempty"`
	Genero            *Genero       `json:"genero,omitempty"`
	FechaNacimiento   *time.Time    `json:"fecha_nacimiento,omitempty"`
	Comuna            *SesionComuna `json:"comuna,omitempty"`
	Direccion         *string       `json:"direccion,omitempty"`
	TelefonoFijo      *string       `json:"telefono_fijo,omitempty"`
	TelefonoMovil     *string       `json:"telefono_movil,omitempty"`
	CorreoElectronico *string       `json:"correo_electronico,omitempty"`
}

// SesionMiembro contiene los datos de contacto si la persona está registrada como miembro de la organización.
type SesionMiembro struct {
	ID                uuid.UUID `json:"id"`
	TelefonoFijo      *string   `json:"telefono_fijo,omitempty"`
	TelefonoMovil     *string   `json:"telefono_movil,omitempty"`
	CorreoElectronico *string   `json:"correo_electronico,omitempty"`
}

// SesionUsuario contiene el identificador del usuario autenticado.
type SesionUsuario struct {
	ID uuid.UUID `json:"id"`
}

// SesionPermiso contiene el identificador y nombres del permiso asignado al rol.
type SesionPermiso struct {
	ID     uuid.UUID `json:"id"`
	Nombre string    `json:"nombre"`
	Slug   string    `json:"slug"`
}

// SesionRol contiene los datos del rol y la lista de permisos dentro de una aplicación.
type SesionRol struct {
	ID       uuid.UUID       `json:"id"`
	Nombre   string          `json:"nombre"`
	Slug     string          `json:"slug"`
	Permisos []SesionPermiso `json:"permisos"`
}

// SesionAplicacion contiene el slug de la aplicación y la lista de roles asociados.
type SesionAplicacion struct {
	AplicacionSlug AplicacionSlug `json:"aplicacion_slug"`
	Roles          []SesionRol    `json:"roles"`
}

// TienePermiso evalúa si la sesión cuenta con el slug de permiso especificado o si ostenta el rol SUPERADMIN.
func (s *Sesion) TienePermiso(permisoSlug string) bool {
	if s == nil {
		return false
	}

	for _, app := range s.Aplicaciones {
		for _, rol := range app.Roles {
			if rol.Slug == "SUPERADMIN" {
				return true
			}
			for _, permiso := range rol.Permisos {
				if permiso.Slug == permisoSlug {
					return true
				}
			}
		}
	}

	return false
}

// TieneAlgunPermiso evalúa si la sesión cuenta con al menos uno de los permisos provistos o rol SUPERADMIN.
func (s *Sesion) TieneAlgunPermiso(permisos ...string) bool {
	if s == nil || len(permisos) == 0 {
		return false
	}

	for _, p := range permisos {
		if s.TienePermiso(p) {
			return true
		}
	}

	return false
}
