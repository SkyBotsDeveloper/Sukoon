// Hindi language file
export const hi = {
  // General
  lang_name: "हिन्दी",
  lang_code: "hi",

  // Start & Help
  start_welcome:
    "नमस्ते! मेरा नाम {bot_name} है - मैं आपके समूहों को प्रबंधित करने में मदद करने के लिए यहां हूं! मुझे पूरी तरह से उपयोग करने के लिए /help का उपयोग करें।\n\nसभी नवीनतम अपडेट के लिए मेरे न्यूज चैनल से जुड़ें।\n\nगोपनीयता नीति देखने के लिए /privacy चेक करें।",
  start_welcome_group: "नमस्ते! मैं {bot_name} हूं, आपका ग्रुप मैनेजमेंट असिस्टेंट। मैं क्या कर सकता हूं यह देखने के लिए /help का उपयोग करें!",
  help_header:
    "नमस्ते! मेरा नाम {bot_name} है। मैं एक ग्रुप मैनेजमेंट बॉट हूं।\n\nमेरे पास कई उपयोगी फीचर्स हैं, जैसे फ्लड कंट्रोल, वार्निंग सिस्टम, नोट्स सिस्टम, और कीवर्ड पर ऑटो रिप्लाई।\n\nउपयोगी कमांड्स:\n- /start: मुझे शुरू करें!\n- /help: यह संदेश भेजता है\n- /donate: डोनेशन जानकारी\n\nसभी कमांड्स / या ! से शुरू होते हैं।",

  // Buttons
  btn_add_to_chat: "अपने चैट में जोड़ें!",
  btn_get_own: "अपना {bot_name} पाएं",
  btn_back: "« वापस",
  btn_support: "सहायता",
  btn_updates: "अपडेट्स चैनल",

  // Admin commands
  admin_promote_success: "{chat} में {user} को सफलतापूर्वक प्रमोट किया!",
  admin_promote_fail: "प्रमोट करने में विफल: {error}",
  admin_demote_success: "{chat} में {user} को सफलतापूर्वक डिमोट किया!",
  admin_demote_fail: "डिमोट करने में विफल: {error}",
  admin_no_reply: "कृपया किसी यूजर को रिप्लाई करें या यूजर ID/यूजरनेम दें।",
  admin_not_admin: "इस कमांड का उपयोग करने के लिए आपको एडमिन होना चाहिए।",
  admin_bot_not_admin: "यह करने के लिए मुझे एडमिन होना चाहिए!",
  admin_cant_self: "आप खुद पर यह नहीं कर सकते!",
  admin_cant_owner: "मैं चैट ओनर पर यह नहीं कर सकता!",
  admin_cant_admin: "मैं दूसरे एडमिन पर यह नहीं कर सकता!",
  admin_list_title: "{chat} में एडमिन:",
  admin_list_creator: "क्रिएटर",
  admin_list_admin: "एडमिन",

  // Moderation - Ban
  ban_success: "{user} को बैन किया!\nकारण: {reason}",
  ban_success_no_reason: "{user} को बैन किया!",
  ban_fail: "बैन करने में विफल: {error}",
  unban_success: "{user} को अनबैन किया!",
  unban_fail: "अनबैन करने में विफल: {error}",

  // Moderation - Mute
  mute_success: "{user} को म्यूट किया!\nअवधि: {duration}\nकारण: {reason}",
  mute_success_no_reason: "{user} को म्यूट किया!\nअवधि: {duration}",
  mute_success_permanent: "{user} को स्थायी रूप से म्यूट किया!\nकारण: {reason}",
  mute_fail: "म्यूट करने में विफल: {error}",
  unmute_success: "{user} को अनम्यूट किया!",
  unmute_fail: "अनम्यूट करने में विफल: {error}",

  // Moderation - Kick
  kick_success: "{user} को किक किया!\nकारण: {reason}",
  kick_success_no_reason: "{user} को किक किया!",
  kick_fail: "किक करने में विफल: {error}",

  // Moderation - Warn
  warn_success: "{user} को वार्न किया ({count}/{max})!\nकारण: {reason}",
  warn_limit_reached: "{user} वार्न लिमिट ({max}) तक पहुंच गया!\nएक्शन: {action}",
  warn_removed: "{user} से एक वार्निंग हटाई। अब उनके पास {count} वार्निंग हैं।",
  warn_reset: "{user} की वार्निंग्स रीसेट!",
  warns_list: "{user} की वार्निंग्स:\n{list}",
  warns_none: "{user} की कोई वार्निंग नहीं है।",

  // Approvals
  approve_success: "{user} को अप्रूव किया! अब वे ऑटो एडमिन एक्शन से बचेंगे।",
  approve_already: "{user} पहले से अप्रूव्ड है!",
  disapprove_success: "{user} को डिसअप्रूव किया!",
  disapprove_not: "{user} अप्रूव्ड नहीं है!",
  approved_list: "{chat} में अप्रूव्ड यूजर्स:\n{list}",
  approved_none: "इस चैट में कोई अप्रूव्ड यूजर नहीं।",

  // Notes
  note_saved: "नोट '{name}' सेव किया!",
  note_deleted: "नोट '{name}' डिलीट किया!",
  note_not_found: "नोट '{name}' नहीं मिला!",
  notes_list: "{chat} में नोट्स:\n{list}",
  notes_none: "इस चैट में कोई नोट नहीं।",

  // Filters
  filter_added: "'{trigger}' के लिए फिल्टर जोड़ा!",
  filter_deleted: "फिल्टर '{trigger}' डिलीट किया!",
  filter_not_found: "फिल्टर '{trigger}' नहीं मिला!",
  filters_list: "{chat} में फिल्टर्स:\n{list}",
  filters_none: "इस चैट में कोई फिल्टर नहीं।",

  // Welcome
  welcome_set: "वेलकम मैसेज सेट!",
  welcome_reset: "वेलकम मैसेज डिफॉल्ट पर रीसेट!",
  welcome_current: "वर्तमान वेलकम मैसेज:\n{message}",
  welcome_on: "वेलकम मैसेज चालू!",
  welcome_off: "वेलकम मैसेज बंद!",
  goodbye_set: "गुडबाय मैसेज सेट!",
  goodbye_reset: "गुडबाय मैसेज डिफॉल्ट पर रीसेट!",
  goodbye_on: "गुडबाय मैसेज चालू!",
  goodbye_off: "गुडबाय मैसेज बंद!",
  default_welcome: "{chat} में आपका स्वागत है, {user}! कृपया /rules पढ़ें।",
  default_goodbye: "अलविदा {user}! हम आपको याद करेंगे।",

  // Rules
  rules_set: "इस चैट के लिए नियम सेट!",
  rules_clear: "नियम हटाए!",
  rules_not_set: "इस चैट के लिए कोई नियम नहीं!",
  rules_title: "{chat} के नियम:",

  // Locks
  lock_success: "{type} लॉक किया!",
  unlock_success: "{type} अनलॉक किया!",
  lock_invalid: "अमान्य लॉक टाइप। वैध टाइप्स: {types}",
  locks_list: "{chat} में वर्तमान लॉक्स:\n{list}",
  lock_message: "इस प्रकार का मैसेज यहां अनुमति नहीं है!",

  // Antiflood
  flood_set: "फ्लड लिमिट {time} सेकंड में {count} मैसेज सेट।",
  flood_off: "एंटीफ्लड बंद।",
  flood_action: "फ्लड एक्शन {action} सेट।",
  flood_triggered: "फ्लडिंग के लिए {user} को {action} किया!",

  // Blocklist
  blocklist_added: "'{word}' ब्लॉकलिस्ट में जोड़ा!",
  blocklist_removed: "'{word}' ब्लॉकलिस्ट से हटाया!",
  blocklist_list: "{chat} में ब्लॉकलिस्टेड शब्द:\n{list}",
  blocklist_none: "इस चैट में कोई ब्लॉकलिस्टेड शब्द नहीं।",
  blocklist_triggered: "मैसेज में ब्लॉकलिस्टेड शब्द था!",

  // Federation
  fed_created: "फेडरेशन '{name}' बनाया!\n\nफेडरेशन ID: `{id}`\n\nइस ID का उपयोग करके अन्य ग्रुप्स आपके फेडरेशन में जुड़ सकते हैं।",
  fed_create_fail: "फेडरेशन बनाने में विफल: {error}",
  fed_exists: "आपके पास पहले से एक फेडरेशन है! पहले /delfed से हटाएं।",
  fed_joined: "यह चैट फेडरेशन: {name} में जुड़ गया\n\nफेडरेशन बैन अब इस चैट पर लागू होंगे।",
  fed_join_fail: "फेडरेशन में जुड़ने में विफल: {error}",
  fed_not_found: "फेडरेशन नहीं मिला!",
  fed_already_joined: "यह चैट पहले से एक फेडरेशन में है!",
  fed_left: "यह चैट फेडरेशन: {name} छोड़ दिया",
  fed_not_in: "यह चैट किसी फेडरेशन में नहीं है।",
  fed_banned: "{user} को फेडरेशन बैन किया!\nफेडरेशन: {fed}\nकारण: {reason}",
  fed_unbanned: "{user} को {fed} से फेडरेशन अनबैन किया!",
  fed_promoted: "{fed} में {user} को फेडरेशन एडमिन बनाया!",
  fed_demoted: "{fed} में {user} को फेडरेशन एडमिन से हटाया!",
  fed_info: "फेडरेशन जानकारी:\n\nनाम: {name}\nID: `{id}`\nओनर: {owner}\nचैट्स: {chats}\nबैन्स: {bans}\nएडमिन्स: {admins}",
  fed_chat_info: "यह चैट इससे जुड़ा है:\n\nफेडरेशन: {name}\nID: `{id}`\nओनर: {owner}",
  fed_admins_list: "{fed} में फेडरेशन एडमिन्स:\n{list}",
  fed_not_admin: "आप फेडरेशन एडमिन नहीं हैं!",
  fed_not_owner: "आप फेडरेशन ओनर नहीं हैं!",
  fed_deleted: "फेडरेशन '{name}' हटा दिया!",
  fed_user_banned: "यह यूजर फेडरेशन बैन है!\nकारण: {reason}\nफेडरेशन: {fed}",

  // AntiRaid
  antiraid_on: "एंटीरेड चालू! नए मेंबर्स को {action} किया जाएगा।",
  antiraid_off: "एंटीरेड बंद!",
  antiraid_action: "एंटीरेड एक्शन {action} सेट।",
  antiraid_triggered: "एंटीरेड: {user} को {action} किया!",

  // CAPTCHA
  captcha_on: "कैप्चा वेरिफिकेशन चालू!",
  captcha_off: "कैप्चा वेरिफिकेशन बंद!",
  captcha_mode: "कैप्चा मोड {mode} सेट।",
  captcha_prompt: "स्वागत है {user}! कृपया नीचे बटन क्लिक करके वेरिफाई करें।",
  captcha_success: "वेरिफिकेशन सफल! {chat} में आपका स्वागत है!",
  captcha_fail: "वेरिफिकेशन विफल! आपको हटा दिया गया।",
  captcha_timeout: "{user} को कैप्चा पूरा न करने पर हटा दिया गया।",

  // Clean Commands
  cleancommands_on: "कमांड डिलीशन चालू! बॉट कमांड्स डिलीट होंगे।",
  cleancommands_off: "कमांड डिलीशन बंद!",

  // Clean Service
  cleanservice_on: "सर्विस मैसेज डिलीशन चालू!",
  cleanservice_off: "सर्विस मैसेज डिलीशन बंद!",

  // Reports
  report_sent: "रिपोर्ट एडमिन्स को भेजी!",
  reports_on: "रिपोर्टिंग चालू!",
  reports_off: "रिपोर्टिंग बंद!",

  // Pin
  pin_success: "मैसेज पिन किया!",
  pin_fail: "पिन करने में विफल: {error}",
  unpin_success: "मैसेज अनपिन किया!",
  unpin_all: "सभी मैसेज अनपिन किए!",

  // Purge
  purge_success: "{count} मैसेज डिलीट किए!",
  purge_fail: "मैसेज डिलीट करने में विफल: {error}",

  // User Info
  info_title: "यूजर जानकारी",
  info_id: "ID: {id}",
  info_first_name: "नाम: {name}",
  info_last_name: "उपनाम: {name}",
  info_username: "यूजरनेम: @{username}",
  info_link: "यूजर लिंक: {link}",
  info_admin: "एडमिन है: {status}",
  info_warns: "वार्निंग्स: {count}",
  info_approved: "अप्रूव्ड: {status}",
  info_gbanned: "ग्लोबली बैन: {status}",

  // Language
  lang_changed: "भाषा {lang} में बदली!",
  lang_current: "वर्तमान भाषा: {lang}",
  lang_list: "उपलब्ध भाषाएं:\n{list}",
  lang_invalid: "अमान्य भाषा कोड! उपलब्ध भाषाओं के लिए /language का उपयोग करें।",

  // Privacy
  privacy_title: "गोपनीयता नीति",
  privacy_text:
    "मैं निम्नलिखित डेटा स्टोर करता हूं:\n- वार्न, अप्रूवल, बैन के लिए यूजर IDs\n- सेटिंग्स के लिए चैट IDs\n- नोट्स और फिल्टर्स के लिए मैसेज कंटेंट\n\nअपना डेटा मांगने के लिए /gdpr का उपयोग करें।\nअपना डेटा हटाने के लिए /deldata का उपयोग करें।",

  // Errors
  error_generic: "एक त्रुटि हुई: {error}",
  error_no_permission: "आपको इसकी अनुमति नहीं है!",
  error_bot_no_permission: "मुझे इसकी अनुमति नहीं है!",
  error_user_not_found: "यूजर नहीं मिला!",
  error_chat_not_found: "चैट नहीं मिला!",
  error_invalid_args: "अमान्य आर्गुमेंट्स! उपयोग: {usage}",
  error_group_only: "यह कमांड केवल ग्रुप्स में उपयोग किया जा सकता है!",
  error_private_only: "यह कमांड केवल प्राइवेट चैट में उपयोग किया जा सकता है!",
  error_admin_only: "यह कमांड केवल एडमिन्स उपयोग कर सकते हैं!",
  error_owner_only: "यह कमांड केवल चैट ओनर उपयोग कर सकता है!",

  // Misc
  id_info: "चैट ID: {chat_id}\nआपकी ID: {user_id}",
  ping_response: "पोंग! {time}ms",
  stats_title: "बॉट आंकड़े",
  stats_users: "यूजर्स: {count}",
  stats_chats: "चैट्स: {count}",
  stats_uptime: "अपटाइम: {time}",
  donate_text: "अगर आप {bot_name} के विकास में सहयोग करना चाहते हैं, यहां डोनेट कर सकते हैं:\n{link}",

  // Connections
  connection_success: "{chat} से कनेक्ट! अब आप PM से मैनेज कर सकते हैं।",
  connection_fail: "कनेक्ट करने में विफल: {error}",
  connection_none: "आप किसी चैट से कनेक्ट नहीं हैं।",
  connection_current: "वर्तमान में कनेक्ट: {chat}",
  disconnect_success: "{chat} से डिस्कनेक्ट!",

  // Log Channel
  log_set: "लॉग चैनल {channel} सेट!",
  log_unset: "लॉग चैनल हटाया!",
  log_current: "वर्तमान लॉग चैनल: {channel}",
  log_none: "कोई लॉग चैनल सेट नहीं।",

  // Disabling
  disable_success: "कमांड {cmd} बंद किया!",
  enable_success: "कमांड {cmd} चालू किया!",
  disabled_list: "{chat} में बंद कमांड्स:\n{list}",
  disabled_none: "इस चैट में कोई बंद कमांड नहीं।",
  cmd_disabled: "यह कमांड इस चैट में बंद है!",

  // Clone
  clone_start: "आपका {bot_name} इंस्टेंस बना रहा हूं...",
  clone_validating: "बॉट टोकन वेरिफाई कर रहा हूं...",
  clone_setting_webhook: "वेबहुक सेट कर रहा हूं...",
  clone_success: "आपका {bot_name} क्लोन तैयार!\n\nबॉट: @{username}\n\nअब आपके पास अपने इंस्टेंस का पूरा कंट्रोल है।",
  clone_fail: "क्लोन बनाने में विफल: {error}",
  clone_invalid_token: "अमान्य बॉट टोकन! BotFather से टोकन लें।",
  clone_instructions:
    "अपना {bot_name} इंस्टेंस बनाने के लिए:\n\n1. BotFather से नया बॉट बनाएं\n2. बॉट टोकन कॉपी करें\n3. PM में /clone <token> भेजें\n\nआपके क्लोन में {bot_name} के सभी फीचर्स होंगे!",
} as const
