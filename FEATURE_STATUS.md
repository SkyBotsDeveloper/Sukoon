# Feature Status

## Working

### Moderation

- `/ban`, `/unban`, `/tban`
- `/mute`, `/unmute`, `/tmute`, `/smute`, `/dmute`
- `/kick`, `/dkick`, `/skick`
- `/warn`, `/warns`, `/resetwarns`
- `/setwarnlimit`, `/setwarnmode`

### Admin And Cleanup

- `/approve`, `/unapprove`, `/approved`
- `/disable`, `/enable`, `/disabled`
- `/logchannel`
- `/reports`, `/report`
- `/cleancommands`
- `/cleanservice`
- `/nocleanservice`
- `/cleanservicetypes`
- `/purge`, `/del`
- `/pin`, `/unpin`, `/unpinall`

### Silent Powers

- `/mod`, `/unmod`
- `/muter`, `/unmuter`
- `/mods`

### Anti-Spam And Verification

- `/lock`, `/unlock`, `/locks`
- `/addblocklist`, `/rmbl`, `/blocklist`
- `/antiflood`
- `/captcha`

### Content And Presence

- `/save`, `/get`, `/clear`
- `/filter`, `/stop`
- `/setwelcome`, `/welcome`
- `/setgoodbye`, `/goodbye`
- `/setrules`, `/rules`
- `/afk`

### Utility And Help

- `/start`
- `/help`
- `/setlang`, `/language`
- `/privacy`, `/mydata`, `/forgetme`

### Owner, Federation, And Clones

- `/broadcast`
- `/stats`
- `/gban`, `/ungban`
- `/bluser`, `/unbluser`
- `/blchat`, `/unblchat`
- `/addsudo`, `/rmsudo`
- `/newfed`, `/delfed`
- `/joinfed`, `/leavefed`
- `/fedinfo`, `/fedadmins`, `/myfeds`
- `/fedpromote`, `/feddemote`
- `/fban`, `/unfban`
- `/fedtransfer`
- `/clone`, `/clone sync`, `/clones`, `/rmclone`

### Policy Features

- antiabuse with narrowed curated matcher
- antibio with exemptions and lease-based checks
- privacy export and delete flows
- per-bot language selection foundation

## Partial

- language support:
  shared translation layer exists, but not every response string has localized variants yet
- rich note and filter formatting:
  implemented structured buttons and rows, but not every historical legacy syntax variant
- metrics:
  observability seam exists, but no external metrics backend is wired by default

## Deferred Or Intentionally Not Claimed

- antiraid
- anti-porn
- night mode
- recurring timed messages
- channel-subscription enforcement
- separate admin web panel
- full Rose-style help or informational command surface beyond the current moderation core
- advanced federation policy toggles that were unsafe or unclear in the legacy runtime

## Final Truth

Anything not listed under `Working` is not complete and should not be treated as production-ready parity.
