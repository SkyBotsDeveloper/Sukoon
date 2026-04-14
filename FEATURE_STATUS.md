# Feature Status

## Working

### Moderation

- `/kickme`
- `/ban`, `/dban`, `/sban`, `/unban`, `/tban`
- `/mute`, `/unmute`, `/tmute`, `/smute`, `/dmute`
- `/kick`, `/dkick`, `/skick`
- `/warn`, `/warns`, `/resetwarns`
- `/setwarnlimit`, `/setwarnmode`

### Admin And Cleanup

- `/approval`, `/approve`, `/unapprove`, `/approved`, `/unapproveall`
- `/promote`, `/demote`
- `/admins`, `/adminlist`, `/admincache`
- `/anonadmin`, `/adminerror`
- `/disable`, `/enable`, `/disableable`, `/disabledel`, `/disableadmin`, `/disabled`
- `/logchannel`, `/setlog`, `/unsetlog`, `/log`, `/nolog`, `/logcategories`
- `/reports`, `/report`
- `/cleancommands`, `/cleancommand`, `/keepcommand`, `/cleancommandtypes`
- `/cleanservice`, `/keepservice`, `/nocleanservice`
- `/cleanservicetypes`
- `/purge`, `/del`
- `/pin`, `/unpin`, `/unpinall`

### Silent Powers

- `/mod`, `/unmod`
- `/muter`, `/unmuter`
- `/mods`

### Anti-Spam And Verification

- `/lock`, `/unlock`, `/locks`, `/lockwarns`, `/locktypes`
- `/allowlist`, `/rmallowlist`, `/rmallowlistall`
- `/addblocklist`, `/rmbl`, `/rmblocklist`, `/unblocklistall`, `/blocklist`
- `/blocklistmode`, `/blocklistdelete`, `/setblocklistreason`, `/resetblocklistreason`
- `/flood`, `/setflood`, `/setfloodtimer`, `/floodmode`, `/setfloodmode`, `/clearflood`
- `/captcha`, `/captchamode`, `/captcharules`, `/captchamutetime`, `/captchakick`, `/captchakicktime`
- `/setcaptchatext`, `/resetcaptchatext`

### Content And Presence

- `/save`, `/notes`, `/saved`, `/get`, `/clear`
- `/filter`, `/filters`, `/stop`, `/stopall`
- `/setwelcome`, `/welcome`
- `/setgoodbye`, `/goodbye`
- `/setrules`, `/resetrules`, `/rules`
- `/connect`, `/disconnect`, `/reconnect`, `/connection`
- `/afk`
- quoted multi-word filter triggers
- contextual fillings for stored content:
  `{first}`, `{last}`, `{fullname}`, `{username}`, `{mention}`, `{id}`, `{chat}`, `{chatname}`, `{rules}`, `{rules:same}`
- random-content separators with `%%%` in notes, filters, welcome, goodbye, and rules text

### Utility And Help

- `/start`
- `/help`
- `/donate`
- `/setlang`, `/language`
- `/privacy`, `/mydata`, `/forgetme`
- callback-driven help pages for:
  admin, approval, bans, antiflood, antiraid, blocklists, captcha, clean commands, disabling, locks, log channels, federations, filters, and formatting
- help subpages for:
  blocklist command examples, federation admin/owner/user commands, filter examples, markdown formatting truth, fillings, random content, buttons, lock descriptions, and lock examples

### Owner, Federation, And Clones

- `/broadcast`
- `/stats`
- `/gban`, `/ungban`
- `/bluser`, `/unbluser`
- `/blchat`, `/unblchat`
- `/addsudo`, `/rmsudo`
- `/newfed`, `/renamefed`, `/delfed`
- `/joinfed`, `/leavefed`
- `/fedinfo`, `/fedadmins`, `/myfeds`, `/chatfed`
- `/fedpromote`, `/feddemote`
- `/feddemoteme`
- `/fban`, `/unfban`
- `/fedtransfer`
- `/clone`, `/clone sync`, `/clones`, `/mybot`, `/rmclone`

### Policy Features

- antiabuse with narrowed curated matcher
- antibio with exemptions, approval bypass, and lease-based checks
- privacy export and delete flows
- per-bot language selection foundation
- callback-driven help and rules UX with in-place message editing

## Partial

- language support:
  shared translation layer exists, but not every response string has localized variants yet
- rich note and filter formatting:
  implemented structured buttons and rows, quoted multi-word filter triggers, contextual fillings, and random content, but not every historical legacy syntax variant
- metrics:
  observability seam exists, but no external metrics backend is wired by default
- help and command-surface parity:
  core moderation, rules, saved content, approvals, connections, PM-guidance UX, and the structured help batches through disabling / federations / filters / formatting are now live, but several long-tail utility families are still intentionally deferred

## Deferred Or Intentionally Not Claimed

- advanced connection features beyond PM management of connected chat content
- anti-porn
- night mode
- recurring timed messages
- channel-subscription enforcement
- private-rules toggle commands
- join-request-specific CAPTCHA delivery
- separate admin web panel
- full help or informational command surface beyond the current scoped live sections
- advanced markdown helper parsing for bold/italics/spoilers/code blocks and note-button syntax in stored content
- advanced federation policy toggles that were unsafe or unclear in the legacy runtime

## Final Truth

Anything not listed under `Working` is not complete and should not be treated as production-ready parity.
