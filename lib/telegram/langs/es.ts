// Spanish language file
export const es = {
  // General
  lang_name: "Español",
  lang_code: "es",

  // Start & Help
  start_welcome:
    "¡Hola! Mi nombre es {bot_name} - ¡Estoy aquí para ayudarte a gestionar tus grupos! Usa /help para descubrir cómo usarme al máximo.\n\nÚnete a mi canal de noticias para recibir información sobre las últimas actualizaciones.\n\nRevisa /privacy para ver la política de privacidad.",
  start_welcome_group:
    "¡Hola! Soy {bot_name}, tu asistente de gestión de grupos. ¡Usa /help para ver lo que puedo hacer!",
  help_header:
    "¡Hola! Mi nombre es {bot_name}. Soy un bot de gestión de grupos.\n\nTengo muchas funciones útiles, como control de flood, sistema de advertencias, sistema de notas, y respuestas automáticas a palabras clave.\n\nComandos útiles:\n- /start: ¡Inícieme!\n- /help: Envía este mensaje\n- /donate: Información de donación\n\nTodos los comandos se pueden usar con: / !",

  // Buttons
  btn_add_to_chat: "¡Añádeme a tu chat!",
  btn_get_own: "Obtén tu propio {bot_name}",
  btn_back: "« Atrás",
  btn_support: "Soporte",
  btn_updates: "Canal de actualizaciones",

  // Admin commands
  admin_promote_success: "¡{user} promovido exitosamente en {chat}!",
  admin_promote_fail: "Error al promover: {error}",
  admin_demote_success: "¡{user} degradado exitosamente en {chat}!",
  admin_demote_fail: "Error al degradar: {error}",
  admin_no_reply: "Por favor responde a un usuario o proporciona un ID/nombre de usuario.",
  admin_not_admin: "Necesitas ser admin para usar este comando.",
  admin_bot_not_admin: "¡Necesito ser admin para hacer esto!",
  admin_cant_self: "¡No puedes hacer esto contigo mismo!",
  admin_cant_owner: "¡No puedo hacer esto al dueño del chat!",
  admin_cant_admin: "¡No puedo hacer esto a otro admin!",
  admin_list_title: "Admins en {chat}:",
  admin_list_creator: "Creador",
  admin_list_admin: "Admin",

  // Moderation - Ban
  ban_success: "¡{user} baneado!\nRazón: {reason}",
  ban_success_no_reason: "¡{user} baneado!",
  ban_fail: "Error al banear: {error}",
  unban_success: "¡{user} desbaneado!",
  unban_fail: "Error al desbanear: {error}",

  // Moderation - Mute
  mute_success: "¡{user} silenciado!\nDuración: {duration}\nRazón: {reason}",
  mute_success_no_reason: "¡{user} silenciado!\nDuración: {duration}",
  mute_success_permanent: "¡{user} silenciado permanentemente!\nRazón: {reason}",
  mute_fail: "Error al silenciar: {error}",
  unmute_success: "¡{user} desilenciado!",
  unmute_fail: "Error al desilenciar: {error}",

  // Moderation - Kick
  kick_success: "¡{user} expulsado!\nRazón: {reason}",
  kick_success_no_reason: "¡{user} expulsado!",
  kick_fail: "Error al expulsar: {error}",

  // Moderation - Warn
  warn_success: "¡{user} advertido ({count}/{max})!\nRazón: {reason}",
  warn_limit_reached: "¡{user} ha alcanzado el límite de advertencias ({max})!\nAcción: {action}",
  warn_removed: "Se eliminó una advertencia de {user}. Ahora tiene {count} advertencias.",
  warn_reset: "¡Advertencias de {user} reiniciadas!",
  warns_list: "Advertencias de {user}:\n{list}",
  warns_none: "{user} no tiene advertencias.",

  // Continue with other translations...
  approve_success: "¡{user} aprobado! Será ignorado por acciones automáticas.",
  approve_already: "¡{user} ya está aprobado!",
  disapprove_success: "¡{user} desaprobado!",
  disapprove_not: "¡{user} no está aprobado!",
  approved_list: "Usuarios aprobados en {chat}:\n{list}",
  approved_none: "No hay usuarios aprobados en este chat.",

  note_saved: "¡Nota '{name}' guardada!",
  note_deleted: "¡Nota '{name}' eliminada!",
  note_not_found: "¡Nota '{name}' no encontrada!",
  notes_list: "Notas en {chat}:\n{list}",
  notes_none: "No hay notas en este chat.",

  filter_added: "¡Filtro para '{trigger}' añadido!",
  filter_deleted: "¡Filtro '{trigger}' eliminado!",
  filter_not_found: "¡Filtro '{trigger}' no encontrado!",
  filters_list: "Filtros en {chat}:\n{list}",
  filters_none: "No hay filtros en este chat.",

  welcome_set: "¡Mensaje de bienvenida establecido!",
  welcome_reset: "¡Mensaje de bienvenida reiniciado!",
  welcome_current: "Mensaje de bienvenida actual:\n{message}",
  welcome_on: "¡Mensajes de bienvenida activados!",
  welcome_off: "¡Mensajes de bienvenida desactivados!",
  goodbye_set: "¡Mensaje de despedida establecido!",
  goodbye_reset: "¡Mensaje de despedida reiniciado!",
  goodbye_on: "¡Mensajes de despedida activados!",
  goodbye_off: "¡Mensajes de despedida desactivados!",
  default_welcome: "¡Bienvenido a {chat}, {user}! Por favor lee las /rules.",
  default_goodbye: "¡Adiós {user}! Te extrañaremos.",

  rules_set: "¡Reglas establecidas para este chat!",
  rules_clear: "¡Reglas eliminadas!",
  rules_not_set: "¡No hay reglas para este chat!",
  rules_title: "Reglas de {chat}:",

  lock_success: "¡{type} bloqueado!",
  unlock_success: "¡{type} desbloqueado!",
  lock_invalid: "Tipo de bloqueo inválido. Tipos válidos: {types}",
  locks_list: "Bloqueos actuales en {chat}:\n{list}",
  lock_message: "¡Este tipo de mensaje no está permitido aquí!",

  flood_set: "Límite de flood establecido a {count} mensajes en {time} segundos.",
  flood_off: "Antiflood desactivado.",
  flood_action: "Acción de flood establecida a {action}.",
  flood_triggered: "¡{user} ha sido {action} por flooding!",

  blocklist_added: "¡'{word}' añadido a la lista negra!",
  blocklist_removed: "¡'{word}' eliminado de la lista negra!",
  blocklist_list: "Palabras en lista negra en {chat}:\n{list}",
  blocklist_none: "No hay palabras en lista negra en este chat.",
  blocklist_triggered: "¡El mensaje contenía una palabra prohibida!",

  fed_created: "¡Federación '{name}' creada!\n\nID de Federación: `{id}`\n\nUsa este ID para que otros grupos se unan.",
  fed_create_fail: "Error al crear federación: {error}",
  fed_exists: "¡Ya tienes una federación! Usa /delfed para eliminarla primero.",
  fed_joined: "Este chat se unió a la federación: {name}\n\nLos baneos de federación ahora aplican a este chat.",
  fed_join_fail: "Error al unirse a la federación: {error}",
  fed_not_found: "¡Federación no encontrada!",
  fed_already_joined: "¡Este chat ya está en una federación!",
  fed_left: "Este chat dejó la federación: {name}",
  fed_not_in: "Este chat no está en ninguna federación.",
  fed_banned: "¡{user} baneado de la federación!\nFederación: {fed}\nRazón: {reason}",
  fed_unbanned: "¡{user} desbaneado de {fed}!",
  fed_promoted: "¡{user} promovido a admin de federación en {fed}!",
  fed_demoted: "¡{user} degradado de admin de federación en {fed}!",
  fed_info:
    "Info de Federación:\n\nNombre: {name}\nID: `{id}`\nDueño: {owner}\nChats: {chats}\nBaneos: {bans}\nAdmins: {admins}",
  fed_chat_info: "Este chat está conectado a:\n\nFederación: {name}\nID: `{id}`\nDueño: {owner}",
  fed_admins_list: "Admins de federación en {fed}:\n{list}",
  fed_not_admin: "¡No eres admin de la federación!",
  fed_not_owner: "¡No eres el dueño de la federación!",
  fed_deleted: "¡Federación '{name}' eliminada!",
  fed_user_banned: "¡Este usuario está baneado de la federación!\nRazón: {reason}\nFederación: {fed}",

  antiraid_on: "¡AntiRaid activado! Los nuevos miembros serán {action}.",
  antiraid_off: "¡AntiRaid desactivado!",
  antiraid_action: "Acción AntiRaid establecida a {action}.",
  antiraid_triggered: "AntiRaid: ¡{user} ha sido {action}!",

  captcha_on: "¡Verificación CAPTCHA activada!",
  captcha_off: "¡Verificación CAPTCHA desactivada!",
  captcha_mode: "Modo CAPTCHA establecido a {mode}.",
  captcha_prompt: "¡Bienvenido {user}! Por favor verifica que eres humano.",
  captcha_success: "¡Verificación exitosa! ¡Bienvenido a {chat}!",
  captcha_fail: "¡Verificación fallida! Has sido eliminado.",
  captcha_timeout: "{user} ha sido eliminado por no completar la verificación.",

  cleancommands_on: "¡Eliminación de comandos activada!",
  cleancommands_off: "¡Eliminación de comandos desactivada!",
  cleanservice_on: "¡Eliminación de mensajes de servicio activada!",
  cleanservice_off: "¡Eliminación de mensajes de servicio desactivada!",

  report_sent: "¡Reporte enviado a los admins!",
  reports_on: "¡Reportes activados!",
  reports_off: "¡Reportes desactivados!",

  pin_success: "¡Mensaje fijado!",
  pin_fail: "Error al fijar: {error}",
  unpin_success: "¡Mensaje desfijado!",
  unpin_all: "¡Todos los mensajes desfijados!",

  purge_success: "¡{count} mensajes eliminados!",
  purge_fail: "Error al eliminar mensajes: {error}",

  info_title: "Info de Usuario",
  info_id: "ID: {id}",
  info_first_name: "Nombre: {name}",
  info_last_name: "Apellido: {name}",
  info_username: "Usuario: @{username}",
  info_link: "Enlace: {link}",
  info_admin: "Es Admin: {status}",
  info_warns: "Advertencias: {count}",
  info_approved: "Aprobado: {status}",
  info_gbanned: "Baneado Globalmente: {status}",

  lang_changed: "¡Idioma cambiado a {lang}!",
  lang_current: "Idioma actual: {lang}",
  lang_list: "Idiomas disponibles:\n{list}",
  lang_invalid: "¡Código de idioma inválido! Usa /language para ver los idiomas disponibles.",

  privacy_title: "Política de Privacidad",
  privacy_text:
    "Almaceno los siguientes datos:\n- IDs de usuario para advertencias, aprobaciones y baneos\n- IDs de chat para configuraciones\n- Contenido de mensajes para notas y filtros\n\nUsa /gdpr para solicitar tus datos.\nUsa /deldata para eliminar tus datos.",

  error_generic: "Ocurrió un error: {error}",
  error_no_permission: "¡No tienes permiso para hacer esto!",
  error_bot_no_permission: "¡No tengo permiso para hacer esto!",
  error_user_not_found: "¡Usuario no encontrado!",
  error_chat_not_found: "¡Chat no encontrado!",
  error_invalid_args: "¡Argumentos inválidos! Uso: {usage}",
  error_group_only: "¡Este comando solo puede usarse en grupos!",
  error_private_only: "¡Este comando solo puede usarse en chat privado!",
  error_admin_only: "¡Este comando solo pueden usarlo admins!",
  error_owner_only: "¡Este comando solo puede usarlo el dueño del chat!",

  id_info: "ID del Chat: {chat_id}\nTu ID: {user_id}",
  ping_response: "¡Pong! {time}ms",
  stats_title: "Estadísticas del Bot",
  stats_users: "Usuarios: {count}",
  stats_chats: "Chats: {count}",
  stats_uptime: "Tiempo activo: {time}",
  donate_text: "Si quieres apoyar el desarrollo de {bot_name}, puedes donar aquí:\n{link}",

  connection_success: "¡Conectado a {chat}! Ahora puedes gestionarlo desde PM.",
  connection_fail: "Error al conectar: {error}",
  connection_none: "No estás conectado a ningún chat.",
  connection_current: "Actualmente conectado a: {chat}",
  disconnect_success: "¡Desconectado de {chat}!",

  log_set: "¡Canal de logs establecido a {channel}!",
  log_unset: "¡Canal de logs eliminado!",
  log_current: "Canal de logs actual: {channel}",
  log_none: "No hay canal de logs establecido.",

  disable_success: "¡Comando {cmd} desactivado!",
  enable_success: "¡Comando {cmd} activado!",
  disabled_list: "Comandos desactivados en {chat}:\n{list}",
  disabled_none: "No hay comandos desactivados en este chat.",
  cmd_disabled: "¡Este comando está desactivado en este chat!",

  clone_start: "Creando tu instancia de {bot_name}...",
  clone_validating: "Validando token del bot...",
  clone_setting_webhook: "Configurando webhook...",
  clone_success:
    "¡Tu clon de {bot_name} está listo!\n\nBot: @{username}\n\nAhora tienes control total sobre tu propia instancia.",
  clone_fail: "Error al crear clon: {error}",
  clone_invalid_token: "¡Token de bot inválido! Obtén un token de BotFather.",
  clone_instructions:
    "Para crear tu propia instancia de {bot_name}:\n\n1. Crea un nuevo bot con BotFather\n2. Copia el token del bot\n3. Envía /clone <token> en PM\n\n¡Tu clon tendrá todas las funciones de {bot_name}!",
} as const
