# Push Ready Checklist

## Repository

- one canonical runtime only
- no obsolete legacy bot-runtime code left
- no superseded phase or audit docs left
- final canonical docs present
- deployment artifacts match the final Go runtime
- CI workflow present

## Validation

- `gofmt -w ./cmd ./internal ./migrations`
- `go test ./...`
- `go test -tags=integration ./...`
- `go build ./cmd/bot-core`
- `go build ./cmd/migrate`

## Operator Validation Still Needed

- live VPS deploy
- live Railway deploy
- live Heroku deploy
- real Telegram webhook registration

## Final Git Steps

Current blocker in this environment:

- `git config user.name` is already set to `SkyBotsDeveloper`
- `git config user.email` is still unset
- GitHub push auth was not available for verification

```powershell
git status
git config user.name "SkyBotsDeveloper"
git config user.email "your-email@example.com"
git add .
git commit -m "Finalize Sukoon Go runtime and clean production handoff"
git push origin main
```
