# i3_next --- Production v1 Release TODO

> هدف: رساندن مخزن از وضعیت فعلی (اسکلت daemon + CLI ابتدایی + stubهای QML) به اولین نسخهٔ قابل نصب، قابل استفاده و قابل پشتیبانی.
>
> **قاعدهٔ تکمیل (Definition of Done):** هر مورد فقط وقتی تیک می‌خورد که ۱) کد پیاده شده، ۲) تست مرتبط (unit/integration/e2e) در CI سبز است، ۳) مستند لازم نوشته شده، ۴) `make check` و `make vm-test` پاس می‌شوند.
>
> **راهنمای علامت‌ها**
>
> | علامت | معنی |
> |---|---|
> | `P0` | Release blocker |
> | `P1` | انتظار می‌رود در v1 باشد |
> | `P2` | در صورت کمبود زمان به v1.x موکول می‌شود |
> | `⛓ X` | وابسته به تسک X |
> | `✅` | معیار پذیرش قابل‌آزمون |
>
> شناسه‌ها: `FIX` اصلاحات فوری، `CORE`، `CFG`، `SWAY`، `API`، `UI`، `SVC`، `DBUS`، `PLG`، `STO`، `SEC`، `OBS`، `SES`، `PKG`، `QA`، `PERF`، `DOC`، `CI`.

---

## 0. محدوده و معیارهای انتشار (انجام‌شده)

- [x] محدودهٔ v1، توزیع‌ها، معماری، ماتریس سازگاری، حداقل سخت‌افزار، مدل نشست (Sway کامل)، فهرست Must-have و Out-of-scope، معیار پذیرش هر قابلیت و سیاست نسخه‌بندی در `docs/RELEASE_SCOPE.md` ثبت شده است.
- [x] هیچ قابلیت README به‌عنوان موجود معرفی نشده مگر پیاده‌شده (بخش "Planned Features").

### بازگشایی موارد ناقص

- [x] **FIX-00a** (P0) سند `RELEASE_SCOPE.md` با واقعیت `go.mod` هم‌خوان نیست: حداقل Go را 1.24 می‌گوید ولی `go.mod` روی `1.27.3` است. ✅ یک نسخهٔ واحد انتخاب شود و `go.mod`، CI، `CONTRIBUTING.md` و `RELEASE_SCOPE.md` یکی شوند. (⛓ FIX-05)

---

## 1. ساختار پروژه و کیفیت پایه

### انجام‌شده
- [x] قراردادهای مشترک (`internal/api/contracts`)، خطاهای دامنه، `LICENSE`، `CONTRIBUTING.md`، `SECURITY.md`، pre-commit، CI پایه، `.golangci.yml`.

### بازگشایی/اصلاحات فوری (Quick wins --- همه P0 مگر ذکر شود)

- [ ] **FIX-01** `make` بدون آرگومان به‌جای `all`، `vm-doctor` را اجرا می‌کند (چون `-include tests/vm/vm.mk` قبل از `all` آمده). ✅ `.DEFAULT_GOAL := all` اضافه شود و تست شود که `make -n` هدف `all` را انتخاب می‌کند. همچنین `$(GO)fmt` با `GOFMT := gofmt` جایگزین شود.
- [ ] **FIX-02** `lib.sh` در ریشهٔ مخزن است ولی `tests/e2e/run.sh` آن را از `tests/e2e/lib.sh` source می‌کند. ✅ فایل به `tests/e2e/lib.sh` منتقل شود؛ `make vm-test` بدون خطای source اجرا شود.
- [ ] **FIX-03** `configs/sway/config` فایل‌های `$HOME/.config/i3-next/sway/*` را include می‌کند که هیچ مرحله‌ای نصب نمی‌کند و keybindingها به دستورات ناموجود (`i3-nextctl launcher`, `i3-nextctl session logout`) اشاره می‌کنند. ✅
  - مسیر include نسبی به فایل config (یا نصب در `~/.config/i3-next/sway/`) شود و در e2e `sway --validate` روی مسیر نصب‌شده پاس شود.
  - تا پیاده‌سازی دستورات، bindهای ناموجود موقتاً به `exec foot`/`swaymsg exit` تغییر کنند یا کامنت شوند.
- [ ] **FIX-04** نسخه در چند جا تکرار شده و ناسازگار است (`i3-nextd`=`0.1.1`، بقیه `0.1.0`، `app.go` رشتهٔ ثابت `"0.1.0"`، `contracts.AppVersion`). ✅ پکیج `internal/version` (Version/Commit/BuildDate از `-ldflags`) ایجاد شود؛ `info.go`ها حذف شوند؛ همهٔ باینری‌ها (از جمله `i3-next-doctor`) `--version` یکسان بدهند؛ e2e `t_version_flags` همهٔ چهار باینری را پوشش دهد.
- [ ] **FIX-05** اعتبارسنجی نسخهٔ Go: ✅ `go.mod` فقط شامل `go` (حداقل واقعی) و در صورت نیاز `toolchain` باشد؛ CI از `go-version-file: go.mod` استفاده کند؛ نسخهٔ انتخابی روی هر دو توزیع Tier-1 قابل نصب باشد (`guest-provision.sh` قبلاً HEAD را چک می‌کند).
- [ ] **FIX-06** در `tests/vm/vm.sh` بررسی checksum ایمیج کامنت شده است. ✅ بررسی بازگردانی شود؛ به‌جای `latest` یک دایرکتوری تاریخ‌دار پین شود تا mismatch ناشی از به‌روزرسانی upstream رخ ندهد؛ در صورت mismatch فایل حذف و خطا داده شود.
- [ ] **FIX-07** `bootstrapDirs` با `0o755` می‌سازد در حالی که `SECURITY.md` permission حداقلی را وعده می‌دهد. ✅ runtime/state/data/config با `0o700` ساخته شوند؛ تست unit روی mode (با `t.TempDir()` و `XDG_*`).
- [ ] **FIX-08** حذف lock file بعد از `flock(UN)` یک race کلاسیک دارد (فرایند دوم روی inode حذف‌شده قفل می‌گیرد). ✅ فایل lock حذف نشود (یا پس از open، inode با `stat` راستی‌آزمایی شود)؛ تست با دو فرایند هم‌زمان که دقیقاً یکی موفق شود.
- [ ] **FIX-09** `internal/app/bootstrap`: ✅
  - `callLifecycle` مبتنی بر reflection با interfaceهای typed (`Starter`, `Stopper`, `HealthChecker`, `Reloader`, `Named`) جایگزین شود؛ (شرط فعلی `param.Implements(ctxType)` جهتش برعکس است).
  - aliasهای `AddService/Add/Use` حذف شوند.
  - `ErrAlreadyRunning/ErrNotRunning` با `internal/core/errors` یکی شوند؛ `ErrLoggerRequired` یک جا بماند.
- [ ] **FIX-10** `App.Run`: ✅
  - خطای `Stop` لاگ و تجمیع شود (نه نادیده گرفته شود).
  - پیام "starting" بعد از موفقیت راه‌اندازی لاگ شود و نسخه از `internal/version` بیاید.
  - شکست سرویس حین اجرا (Health/exit) مانیتور شود و طبق policy باعث degraded یا shutdown شود.
  - تست‌ها: rollback هنگام شکست start، ترتیب stop معکوس، timeout در shutdown، hook failure، اجرای دوباره پس از Run.
- [ ] **FIX-11** `.gitignore`: الگوی `tmp*` بیش‌ازحد گسترده است (فایل‌های مثل `tmpl/` را ignore می‌کند) → `/tmp/`. ✅
- [ ] **FIX-12** (P0) CI: ✅
  - actionها با SHA پین شوند؛ `staticcheck` و `golangci-lint` به نسخهٔ مشخص پین شوند (نه `latest`).
  - `govulncheck ./...` اضافه شود.
  - `concurrency: cancel-in-progress` و `timeout-minutes` برای jobها.
  - job QML با `|| true` همیشه سبز می‌شود؛ باید در نبود `qmllint` واقعاً نصب یا job fail شود.
  - ماتریس Go (حداقل و فعلی).
- [ ] **FIX-13** `README.md` بخش "Structure" ساختار هدف است نه واقعی؛ ✅ عنوان "Target structure (not yet created)" بخورد یا ساختار واقعی نمایش داده شود، و آیتم‌های `[x]` بخش ۱ که فقط اسکلت‌اند (ساختار پکیج‌ها، فایل‌های خالی config/QML) به حالت `[ ]` برگردند تا با کار واقعی تیک بخورند.
- [ ] **FIX-14** (P1) سند `docs/SECURITY.md` در `SECURITY.md` ارجاع داده شده ولی وجود ندارد؛ `scripts/build.sh` منبع build دوم است؛ `scripts/install.sh` فقط باینری نصب می‌کند (نه config/QML/session). ✅ تصمیم نهایی و هم‌خوانی (⛓ PKG-01).
- [ ] **FIX-15** `i3-next-doctor` همیشه exit 0 می‌دهد حتی با `[FAIL]`. ✅ exit code غیرصفر در FAIL، `--json`، و تست e2e جدید (⛓ OBS-04).
- [ ] **FIX-16** (P1) قرارداد نام‌گذاری یکدست: `i3_next` (module/AppName) در برابر `i3-next` (dirها/باینری‌ها). ✅ در `docs/development/naming.md` ثبت شود و اسکریپتی بررسی کند.

### باقی‌ماندهٔ بخش ۱
- [ ] **STRUCT-01** (P0) پکیج‌ها را فقط وقتی بساز که اولین کد واقعی دارند (بدون پوشهٔ خالی). ✅ قاعدهٔ import (core ⟵ adapters/services ⟵ api) با `depguard`/`go-arch-lint` در `.golangci.yml` اجباری شود.
- [ ] **STRUCT-02** (P1) `CODEOWNERS`, قالب PR/Issue، `CHANGELOG.md` (Keep a Changelog).
- [ ] **STRUCT-03** (P1) `go.sum` و سیاست dependency: هر dependency جدید با دلیل در PR ثبت و مجوزش بررسی شود (⛓ SEC-10).

---

## 2. Desktop Core و چرخه عمر

### 2.1 Bootstrap و lifecycle
- [ ] **CORE-01** (P0) ترتیب راه‌اندازی deterministic: سرویس‌ها با `DependsOn()` و مرتب‌سازی توپولوژیک؛ چرخه تشخیص و خطا بدهد. ✅ تست ترتیب و تشخیص cycle. ⛓ FIX-09
- [ ] **CORE-02** (P0) قرارداد سرویس در `internal/core/registry`:
  ```go
  type Service interface {
      Name() string
      Start(ctx context.Context) error
      Stop(ctx context.Context) error
      Health(ctx context.Context) contracts.ServiceInfo
  }
  ```
  `Reloader` اختیاری؛ سرویس‌ها `required` یا `optional` باشند (optional ⟵ degraded). ✅
- [ ] **CORE-03** (P0) Rollback مرحله‌ای: شکست سرویس required ⟵ توقف همهٔ سرویس‌های شروع‌شده؛ شکست optional ⟵ `degraded` و ادامه. ✅ تست با fake service.
- [ ] **CORE-04** (P0) Shutdown ترتیبی با timeout هر سرویس (نه فقط کلی)؛ سرویس گیرکرده kill/لاگ شود. ✅ تست hang.
- [ ] **CORE-05** (P0) تست سیگنال: SIGTERM/SIGINT/SIGHUP(=reload) و exit code مشخص (e2e `t_daemon_lifecycle` موجود گسترش یابد).
- [ ] **CORE-06** (P1) Single-instance: پیام خطای قابل‌فهم + PID فرایند قبلی؛ stale lock تشخیص داده شود. ⛓ FIX-08
- [ ] **CORE-07** (P1) وضعیت daemon و علت آخرین شکست در state (`status` API) و فایل `state_dir/last-failure.json`.

### 2.2 Process supervisor (`internal/app/supervisor`)
- [ ] **CORE-10** (P0) اجرای process با `exec.CommandContext` و آرگومان‌ها (بدون shell)؛ مسیر با `LookPath` و بررسی owner/permission اعتبارسنجی شود. ✅
- [ ] **CORE-11** (P0) گرفتن stdout/stderr به log ساختاریافته (با خط‌بندی و سقف اندازه)، ثبت exit code/signal. ✅
- [ ] **CORE-12** (P0) Restart policy: `max_restarts` در `window`، backoff نمایی با jitter؛ فراتر از سقف ⟵ `failed/degraded` بدون loop. ✅ تست با process دائماً کرشی.
- [ ] **CORE-13** (P0) توقف child: SIGTERM ⟵ grace ⟵ SIGKILL روی process group (`Setpgid`)؛ بعد از مرگ daemon هیچ یتیمی نماند (`Pdeathsig`). ✅ تست e2e.
- [ ] **CORE-14** (P1) Environment allowlist برای child و redact متغیرهای حساس (`*TOKEN*`, `*SECRET*`, `*PASSWORD*`) در log.

### 2.3 Event bus و state
- [ ] **CORE-20** (P0) `internal/core/eventbus` با generics: `Subscribe[T](topic) (<-chan T, cancel)`. ✅
- [ ] **CORE-21** (P0) سیاست backpressure مستند و تست‌شده: صف bounded؛ برای eventهای state (آخرین مقدار مهم است) coalesce، برای eventهای ضروری drop نشود/ خطا گزارش شود؛ panic در handler جداسازی شود.
- [ ] **CORE-22** (P0) تعریف eventهای اصلی در `internal/api/events` با schema نسخه‌دار: `SessionStarted`, `SwayConnected/Disconnected`, `WorkspaceChanged`, `OutputChanged`, `ConfigChanged`, `ServiceStateChanged`, `ShutdownRequested`.
- [ ] **CORE-23** (P0) `internal/core/state`: snapshot immutable + نسخهٔ monotonic (برای بازیابی snapshot در API)؛ تست `-race` و بنچمارک. ⛓ API-09
- [ ] **CORE-24** (P1) تست بار (۱۰k event/s، ۱۰۰ subscriber) و بررسی نشت goroutine با `goleak`.

---

## 3. Configuration (`internal/core/config`)

> فعلاً فقط `paths.go` وجود دارد؛ همهٔ موارد زیر باز هستند.

- [ ] **CFG-01** (P0) ساختارهای Go تایپ‌شده برای چهار فایل + `version` و parser سخت‌گیر (`yaml.v3` با `KnownFields(true)`) — dependencyها ثبت شوند. ✅ فیکسچرهای valid/invalid.
- [ ] **CFG-02** (P0) schema نوشته‌شده (JSON Schema یا validator Go) و پیام خطا با مسیر کلید و شمارهٔ خط (`desktop.yaml:12 ui.theme: unknown value "x"`).
- [ ] **CFG-03** (P0) `//go:embed` برای `configs/defaults/*` و پر کردن مقادیر معتبر (`shortcuts.yaml`/`plugins.yaml` نباید فقط لیست خالی باشند: حداقل shortcutهای پیش‌فرض واقعی). 
- [ ] **CFG-04** (P0) اولویت لایه‌ها مستند و پیاده: defaults < `/etc/i3-next` < `~/.config/i3-next` < env (`I3NEXT_*`) < flags. ✅ تست merge.
- [ ] **CFG-05** (P0) نصب defaults در اولین اجرا با `O_EXCL` (هرگز overwrite). ✅ تست idempotent.
- [ ] **CFG-06** (P0) نوشتن atomic: temp در همان دایرکتوری ⟵ `fsync` ⟵ `rename` ⟵ `fsync(dir)`؛ permission `0600/0644` مشخص. ✅ تست crash-injection.
- [ ] **CFG-07** (P0) config نامعتبر: daemon با آخرین config معتبر (یا defaults) بالا بیاید، وضعیت `degraded` + خطای actionable؛ بدون restart-loop. ✅ e2e.
- [ ] **CFG-08** (P1) Migration نسخه‌دار (`vN ⟶ vN+1`) با backup خودکار قبل از migrate و تست golden-file.
- [ ] **CFG-09** (P1) Reload امن (SIGHUP + `config.reload` API): diff محاسبه شود؛ کلیدهای non-reloadable صریحاً گزارش شوند.
- [ ] **CFG-10** (P1) `i3-nextctl config validate|show|backup|restore`. ⛓ OBS-05
- [ ] **CFG-11** (P1) `docs/config/` مرجع کامل کلیدها (تولید خودکار از schema) + نمونه.
- [ ] **CFG-12** (P1) File watcher (`fsnotify`) اختیاری با debounce برای reload.

---

## 4. Sway Integration (`internal/adapters/sway`)

### 4.1 IPC client
- [ ] **SWAY-01** (P0) Client مستقل: تشخیص `SWAYSOCK` (و fallback `swaymsg -t get_socket_path` فقط اگر لازم)، خطای واضح در نبود. ✅
- [ ] **SWAY-02** (P0) پروتکل i3-ipc: magic `i3-ipc`، header (len,type) با endianness native، تابع framing تست‌شده با fuzz (`go test -fuzz`) برای پیام‌های ناقص/بزرگ.
- [ ] **SWAY-03** (P0) Commands (`RUN_COMMAND`)، queries (`GET_WORKSPACES/OUTPUTS/TREE/INPUTS/VERSION`) و subscribe (window/workspace/output/mode/shutdown/input). 
- [ ] **SWAY-04** (P0) Reconnect با backoff + resubscribe + بازسازی snapshot کامل پس از اتصال؛ eventهای `shutdown` از Sway درست مدیریت شوند. ✅ e2e: kill/restart Sway.
- [ ] **SWAY-05** (P0) Context/timeout برای هر درخواست؛ همزمانی ایمن (یک reader goroutine، correlation بر اساس نوع پیام).
- [ ] **SWAY-06** (P0) مدل‌های typed + fixture JSON واقعی از Sway 1.8 و 1.10+ در `internal/adapters/sway/testdata/`. ✅ تست parse با فیلدهای ناشناخته‌ی tolerated.
- [ ] **SWAY-07** (P1) `Adapter` interface در `internal/adapters/sway` (سرویس‌ها فقط به آن وابسته‌اند) + fake برای تست سرویس‌ها.
- [ ] **SWAY-08** (P1) Feature detection بر پایهٔ نسخه (جدول قابلیت در کد + مستند).
- [ ] **SWAY-09** (P0) Integration test با Sway headless در `make vm-test` (همان زیرساخت `tests/e2e`).

### 4.2 مدیریت پنجره/workspace
- [ ] **SWAY-20** (P0) سرویس `workspace`/`window-manager`: snapshot + eventها ⟵ event bus (⛓ CORE-20/22).
- [ ] **SWAY-21** (P0) عملیات: focus/move/close/floating/layout/move-to-workspace/output. ورودی‌ها (نام workspace، app_id) **escape/validate** شوند تا injection به command Sway ممکن نباشد. ✅ تست با ورودی `; exec ...`.
- [ ] **SWAY-22** (P1) چند‌مانیتوره: workspace↔output، hotplug (تست موجود `sway_monitor_hotplug` به سرویس وصل شود).
- [ ] **SWAY-23** (P1) Window rules با schema؛ تولید `window-rules` از config به‌جای فایل ثابت.
- [ ] **SWAY-24** (P0) تولید config Sway از template (`text/template`) در `state_dir/generated/`، اعتبارسنجی با `sway --validate -c`، و **هرگز** overwrite config کاربر بدون backup/رضایت. ✅
- [ ] **SWAY-25** (P1) مستند keybindingهای پیش‌فرض و راه بازگشت به config پیش‌فرض.

---

## 5. API بین Core و Shell (`internal/api/rpc`)

- [ ] **API-01** (P0) تصمیم ثبت‌شده (ADR در `docs/architecture/adr/0001-rpc.md`): Unix socket + JSON-RPC 2.0 (line- یا length-prefixed). 
- [ ] **API-02** (P0) Socket در `$XDG_RUNTIME_DIR/i3-next/i3-next.sock` با `0600`؛ دایرکتوری `0700`؛ `umask` قبل از `Listen`؛ حذف stale socket فقط وقتی connect ناموفق باشد. ✅ تست permission.
- [ ] **API-03** (P0) بررسی `SO_PEERCRED`: فقط UID صاحب نشست. ✅ تست رد UID دیگر (در VM با کاربر دوم).
- [ ] **API-04** (P0) handshake `hello {api_version, client}` و پاسخ سازگاری/ناسازگاری (`contracts.APIVersion`)؛ خطای `version_mismatch` به UI نمایش داده شود.
- [ ] **API-05** (P0) محدودیت‌ها: حداکثر اندازهٔ پیام (مثلاً 1 MiB)، timeout هر درخواست، سقف درخواست هم‌زمان، سقف اتصال. ✅ تست fuzz/over-limit.
- [ ] **API-06** (P0) نگاشت خطا: `internal/core/errors` ⟷ `contracts.ErrorCode` در یک جا (جدول واحد) با تست.
- [ ] **API-07** (P0) متدهای حداقلی v1: `status.get`, `service.list/restart`, `config.get/validate/reload`, `workspace.*`, `output.*`, `launcher.search/launch`, `notification.*`, `session.logout/restart/shutdown`.
- [ ] **API-08** (P0) Event stream: `events.subscribe` (notification بدون id) با شمارهٔ ترتیب.
- [ ] **API-09** (P0) بازیابی پس از reconnect: `state.snapshot` با `version`؛ client با gap تشخیص می‌دهد و snapshot جدید می‌گیرد. ⛓ CORE-23
- [ ] **API-10** (P1) مستندات `docs/api/` (تولید از کد) + نمونه `socat`.
- [ ] **API-11** (P1) Contract test مشترک: فایل‌های golden JSON که هم Go و هم QML (mock backend) روی آن تست می‌شوند.
- [ ] **API-12** (P1) Rate limit روی متدهای حساس (shutdown/launch).

---

## 6. Quickshell Desktop Shell

### 6.1 زیرساخت
- [ ] **UI-01** (P0) `Backend.qml` واقعی: اتصال به socket (Quickshell `Socket` / `SplitParser`)، handshake، reconnect با backoff، وضعیت `disconnected/connecting/connected/incompatible`. ⛓ API-04
- [ ] **UI-02** (P0) `EventBridge.qml`: parse eventها و پر کردن modelهای QML (`ListModel`/`QtObject`) بدون block کردن thread UI.
- [ ] **UI-03** (P0) جریان راه‌اندازی `Main→Shell→Session` واقعی و اتصال `qmldir` (ماژول `i3_next` با import path قابل‌کشف، نصب در `/usr/share/i3-next/quickshell`).
- [ ] **UI-04** (P0) `Variants`/`Scope` برای هر مانیتور؛ hotplug درست (panel ساخته/حذف شود). ✅ e2e با `create_output`/`unplug`.
- [ ] **UI-05** (P0) مدیریت خطا: شکست یک component کل shell را نبرد (Loader با `onStatusChanged`)؛ crash shell ⟵ supervisor ری‌استارت (⛓ CORE-12).
- [ ] **UI-06** (P1) Theme tokens جدا (`themes/default/Tokens.qml`) و تغییر زنده از `Settings.qml`.
- [ ] **UI-07** (P1) Logging/debug mode (`I3NEXT_UI_DEBUG=1`) و ارسال خطاها به daemon.
- [ ] **UI-08** (P1) Scale/رزولوشن‌های مختلف؛ دسترس‌پذیری (focus, keyboard-nav, contrast ≥ 4.5:1، حداقل اندازهٔ فونت).
- [ ] **UI-09** (P0) `qmllint` در CI اجباری و بدون خطا؛ تست smoke: `quickshell -p ui/quickshell` در VM بدون خطا بالا بیاید.

### 6.2 قابلیت‌ها (ترتیب پیشنهادی)
- [ ] **UI-10** (P0) Panel: ساعت، workspaces، شبکه، صدا، باتری (در نبود باتری مخفی). ⛓ SVC-*
- [ ] **UI-11** (P0) Launcher: خواندن desktop entryها (XDG، `NoDisplay/Hidden/OnlyShowIn`)، جست‌وجوی fuzzy، اجرای امن (بدون shell؛ field codes `%u %f` درست پردازش شوند). 
- [ ] **UI-12** (P0) Session menu: logout/restart/shutdown با تأیید.
- [ ] **UI-13** (P1) Notification popup + Notification Center (⛓ SVC-03).
- [ ] **UI-14** (P1) Control Center (audio، brightness، network، Bluetooth در صورت وجود).
- [ ] **UI-15** (P1) OSD (volume/brightness/workspace).
- [ ] **UI-16** (P1) Task/window list.
- [ ] **UI-17** (P1) حالت‌های empty/loading/error/disconnected برای همهٔ view‌ها (چک‌لیست در PR template).
- [ ] **UI-18** (P1) Settings پایهٔ ظاهر/theme.

---

## 7. سرویس‌های سیستم

### 7.1 الزامات مشترک (قبل از هر سرویس)
- [ ] **SVC-00** (P0) قالب مشترک سرویس (`internal/services/<name>`): config `enabled`، dependency اختیاری ⟵ `degraded` با دلیل، timeout/cancel روی عملیات، انتشار `ServiceStateChanged`، تست با fake adapter. ✅ تست «dependency غایب».

### 7.2 Must-have
- [ ] **SVC-01** (P0) Session manager (⛓ SES-*).
- [ ] **SVC-02** (P1) Wallpaper: `swaybg` تحت supervisor، چندمانیتوره.
- [ ] **SVC-03** (P1) Notification: پیاده‌سازی `org.freedesktop.Notifications` روی session bus (اگر daemon دیگری نام را دارد خطای واضح)، پشتیبانی actions/urgency/timeout، سقف تعداد و اندازه. ⛓ DBUS-01
- [ ] **SVC-04** (P1) Audio: PipeWire/WirePlumber (`wpctl` با آرگومان لیست یا کتابخانه)، volume/mute/default device، رویداد تغییر.
- [ ] **SVC-05** (P1) Network: NetworkManager D-Bus (status فقط‌خواندنی در v1؛ نبود NM ⟵ degraded).
- [ ] **SVC-06** (P1) Power: UPower (باتری/AC) + logind (suspend/reboot/poweroff) با polkit.
- [ ] **SVC-07** (P0) Lock screen: `swaylock` (ابزار آزموده) ⟵ بدون lock «نمایشی». رفتار crash مولفه: **fail-closed** (نشست باز نماند؛ `ext-session-lock` ⟵ در crash کلاینت صفحه قفل می‌ماند). ✅ تست دستی/خودکار با kill swaylock.
- [ ] **SVC-08** (P1) Idle: `swayidle` با policy از config (lock/DPMS/suspend) و inhibit.
- [ ] **SVC-09** (P1) Screenshot: `grim`/`slurp`، ذخیره در `XDG_PICTURES_DIR`، نام فایل امن.
- [ ] **SVC-10** (P1) Clipboard: `wl-clipboard` (get/set در محدودهٔ v1، بدون history پایدار).
- [ ] **SVC-11** (P1) Display: خروجی‌ها/رزولوشن/scale با اعتبارسنجی مود (از `get_outputs`) و revert خودکار پس از N ثانیه (مثل «Keep changes?»).
- [ ] **SVC-12** (P1) Media: MPRIS روی D-Bus.
- [ ] **SVC-13** (P1) Brightness (برای Control Center): `brightnessctl`/logind `SetBrightness`.

### 7.3 قابل تعویق (P2)
- [ ] Bluetooth کامل، File search/command palette، Clipboard history، Layout manager، Color picker، System monitor، Workspace overview/انیمیشن، Power profiles.

---

## 8. D-Bus و یکپارچگی سیستم (`internal/adapters/dbus`)

- [ ] **DBUS-01** (P0/P1) انتخاب کتابخانه (`godbus/dbus/v5`) + wrapper با reconnect/timeout و تفکیک session/system bus. جدول «چه سرویسی از کدام bus» در `docs/architecture/dbus.md`.
- [ ] **DBUS-02** (P1) Fake bus برای تست (یا `dbus-run-session` در e2e).
- [ ] **DBUS-03** (P1) ورودی‌های D-Bus اعتبارسنجی شوند (اندازه/نوع/UTF-8).
- [ ] **DBUS-04** (P1) polkit/logind برای power: حداقل دسترسی، بدون `sudo`.
- [ ] **DBUS-05** (P2) D-Bus service file برای activation (در صورت نیاز) (⛓ PKG-05).
- [ ] **DBUS-06** (P1) تست در نشست ناقص (بدون UPower/NM/…).

---

## 9. Plugin System

- [ ] **PLG-00** (P0) **تصمیم نهایی v1:** پیشنهاد = سیستم plugin خارجی در v1 **غیرفعال**؛ فقط built-in. ثبت در ADR و `RELEASE_SCOPE.md`. ✅ کد لود third-party در build v1 وجود نداشته باشد (یا پشت build tag `plugins`).
- [ ] **PLG-01** (P2) اگر فعال شد: manifest نسخه‌دار + schema، permission model، اعتماد/امضا، resource limit، sandbox (bubblewrap/landlock/seccomp) با مرز امنیتی واقع‌بینانه مستند.
- [ ] **PLG-02** (P2) SDK و plugin نمونه، تست plugin مخرب/خراب، تست سازگاری API.
- [ ] **PLG-03** (P1) `plugins.yaml` در v1 اگر پر شد: plugin ناشناس = رد با پیام واضح.

---

## 10. Storage

- [ ] **STO-00** (P0) تصمیم: کدام داده‌ها persistent‌اند؟ (پیشنهاد v1: فقط state کوچک؛ شاید حتی بدون SQLite و با JSON atomic). اگر SQLite لازم نیست، آن را از README/معماری حذف یا به v1.x منتقل کن.
- [ ] **STO-01** (P1) اگر لازم: SQLite pure-Go (`modernc.org/sqlite`، بدون CGO)، WAL، `0600`، migration نسخه‌دار (`internal/storage/migrations`)، repository interface.
- [ ] **STO-02** (P1) خرابی DB/migration ناموفق: backup خودکار، `quarantine`، راه‌اندازی با DB تازه + هشدار؛ تست.
- [ ] **STO-03** (P1) سیاست retention/پاک‌سازی cache و history؛ بدون ذخیرهٔ داده حساس.
- [ ] **STO-04** (P1) backup/restore: `i3-nextctl backup|restore` (config + state).

---

## 11. Security و Privacy

- [ ] **SEC-01** (P0) `docs/SECURITY.md` (threat model): دارایی‌ها، مرزهای اعتماد (socket، D-Bus، config، child processes، plugin)، STRIDE خلاصه، فرضیات.
- [ ] **SEC-02** (P0) بازبینی permission همهٔ فایل/socket/lock (⛓ FIX-07, API-02) با تست خودکار.
- [ ] **SEC-03** (P0) Path traversal/symlink: باز کردن فایل‌ها با `os.Root`/`openat2(RESOLVE_BENEATH)` یا بررسی `EvalSymlinks` + owner؛ برای config/asset/desktop-entry.
- [ ] **SEC-04** (P0) Lint امنیتی: `gosec` فعال است؛ نتایج بدون `//nolint` بی‌دلیل. اضافه‌کردن `govulncheck` (⛓ FIX-12).
- [ ] **SEC-05** (P0) Input validation مرکزی برای IPC/D-Bus/config؛ fuzz روی parserها.
- [ ] **SEC-06** (P0) Logها بدون secret/مسیر شخصی (redaction helper + تست).
- [ ] **SEC-07** (P0) Lock fail-closed (⛓ SVC-07).
- [ ] **SEC-08** (P1) Hardening systemd user unit (`NoNewPrivileges`, `ProtectSystem`, … در حد سازگار با نشست).
- [ ] **SEC-09** (P1) Supply chain: پین کردن actions/ابزارها، `go mod verify`، SBOM، امضای release (cosign/minisign)، build قابل تکرار (`-trimpath` موجود است).
- [ ] **SEC-10** (P1) `THIRD_PARTY_LICENSES.md` (تولید با `go-licenses`) و بررسی سازگاری مجوز.
- [ ] **SEC-11** (P0) سیاست telemetry: **هیچ** (مستند در README/PRIVACY.md).
- [ ] **SEC-12** (P1) فرایند افشای آسیب‌پذیری: ایمیل/Private reporting واقعاً فعال باشد (در `SECURITY.md` هنوز «TBD‌گونه» است).

---

## 12. Logging، Diagnostics، Observability

- [ ] **OBS-01** (P0) Logger: فیلدهای ثابت `component`, `version`, `commit`, `session_id`؛ خروجی به journald (stderr با فرمت journald یا `slog` handler) در نشست واقعی و فایل در `state_dir/logs/` در حالت dev؛ rotation/سقف (`lumberjack` یا تکیه بر journald).
- [ ] **OBS-02** (P1) سطح log از config و runtime (`i3-nextctl log-level`)؛ `AddSource` فقط debug (موجود است).
- [ ] **OBS-03** (P1) خطاهای actionable (کلید `hint=`) جدا از خطاهای عادی.
- [ ] **OBS-04** (P0) `i3-next-doctor` بازنویسی: چک‌ها با ساختار `{id,status,detail,remedy}`؛ پوشش: Sway(+نسخه)، Quickshell(+نسخه)، Qt6، Wayland env، D-Bus session/system، ابزارهای اختیاری (swaylock, grim, wpctl, …)، permission دایرکتوری‌ها، config معتبر، socket API؛ `--json`؛ exit code (0 OK، 1 WARN با `--strict`، 2 FAIL). ✅ e2e `t_doctor_*`.
- [ ] **OBS-05** (P0) `i3-nextctl`: کلاینت واقعی API؛ دستورات `status`, `stop`, `restart`, `config validate`, `services`, `log-level`؛ خروجی `--json`؛ exit code استاندارد؛ دستور فعلی `status` که رشتهٔ ثابت چاپ می‌کند حذف شود. ✅ e2e: با daemon در حال اجرا `status` ⟵ `running`.
- [ ] **OBS-06** (P1) `i3-nextctl diagnostics bundle` با redact مسیر/نام کاربر/IP.
- [ ] **OBS-07** (P2) متریک داخلی ساده (counter/gauge در `status`) بدون export شبکه.

---

## 13. Session، Login و مدیریت نشست

- [ ] **SES-01** (P0) `packaging/wayland-sessions/i3-next.desktop` (`Exec=i3-next-session`, `Type=Application`).
- [ ] **SES-02** (P0) `i3-next-session` واقعی: تنظیم env (`XDG_CURRENT_DESKTOP=i3-next:sway`, `XDG_SESSION_TYPE=wayland`, `XDG_SESSION_DESKTOP`, `MOZ_ENABLE_WAYLAND`, `QT_QPA_PLATFORM`…)، `dbus-update-activation-environment --systemd`، `systemctl --user import-environment`، اجرای Sway، و سپس `i3-nextd`/Quickshell با ترتیب مشخص. استفاده از `systemd --user` targetها (`i3-next-session.target`) توصیه می‌شود. ✅
- [ ] **SES-03** (P0) جلوگیری از اجرای تکراری (lock/`systemctl --user` unit واحد).
- [ ] **SES-04** (P0) خطای startup: پیام، لاگ در `state_dir/logs/session-*.log`، fallback به یک TTY/foot و مستند recovery. ✅ تست با daemon خراب.
- [ ] **SES-05** (P0) Logout تمیز: توقف سرویس‌ها (ترتیب معکوس)، `swaymsg exit`، پاک‌سازی socket/lock.
- [ ] **SES-06** (P1) Restart shell بدون logout (`i3-nextctl shell restart`).
- [ ] **SES-07** (P1) Suspend/resume و lock-before-sleep (logind inhibitor)؛ تست دستی در QA.
- [ ] **SES-08** (P1) مستند اجرای dev (nested Sway) و اجرای واقعی (DM).
- [ ] **SES-09** (P0) تست VM: `make vm-gui` سشن کامل (autologin) بدون دخالت دستی بالا بیاید و اسکرین‌شات ثبت شود.

---

## 14. Packaging و نصب

### 14.1 Build/Release
- [ ] **PKG-01** (P0) `make build` تنها منبع build؛ `scripts/build.sh` حذف یا فقط wrapper؛ `make install DESTDIR=… PREFIX=…` استاندارد (باینری، configs، QML، session، docs).
- [ ] **PKG-02** (P0) ldflags یکسان از `internal/version` (⛓ FIX-04). ✅ `i3-nextd --version` شامل commit و date واقعی (نه `dev/unknown`) در release.
- [ ] **PKG-03** (P1) build debug/release جدا؛ `-trimpath`، `CGO_ENABLED=0` در release.
- [ ] **PKG-04** (P1) SBOM (`syft`/`cyclonedx-gomod`)، checksum، امضا.

### 14.2 بسته‌ها
- [ ] **PKG-10** (P0) `.deb` برای Ubuntu 24.04 و Debian 13 (`packaging/debian/`): `Depends` (sway, quickshell\*, qt6 runtime, dbus-user-session, …)، `Recommends` (pipewire, network-manager, upower, swaylock, swayidle, grim, slurp, wl-clipboard)، `lintian` تمیز. ✅ نصب روی VM تمیز هر دو توزیع. (\*Quickshell ممکن است در Ubuntu 24.04 نباشد ⟵ تصمیم: PPA/backport/vendoring و ثبت در docs.)
- [ ] **PKG-11** (P0) postinst/prerm: بدون لمس config کاربر؛ upgrade حافظ config؛ purge فقط با رضایت.
- [ ] **PKG-12** (P1) systemd user units و dbus/polkit فایل‌ها.
- [ ] **PKG-13** (P2) PKGBUILD (Arch، best-effort).
- [ ] **PKG-14** (P1) آزمون خودکار نصب/ارتقا/حذف روی VM (⛓ QA-05).
- [ ] **PKG-15** (P1) تصمیم و مستند: نصب از repository (APT repo/Release assets) یا آفلاین.

### 14.3 Scriptها
- [ ] **PKG-20** (P1) `scripts/dev-setup.sh` idempotent؛ `scripts/install.sh` با `--prefix`، `--dry-run`، بررسی پیش‌نیاز و **فقط فایل‌های متعلق** (manifest نصب)؛ `uninstall.sh` از همان manifest. ✅ shellcheck + تست مسیر با فاصله.
- [ ] **PKG-21** (P1) هیچ اسکریپتی فایل کاربر را بدون backup overwrite نکند.

---

## 15. Testing و QA

### 15.1 Go
- [ ] **QA-01** (P0) تست unit برای هر پکیج جدید (هدف: مسیرهای حیاتی core/config/api/sway ≥ 80% line، بیشتر تمرکز روی رفتار).
- [ ] **QA-02** (P0) `go test -race -count=1 ./...` + `goleak`.
- [ ] **QA-03** (P1) Fuzz: IPC framing، JSON-RPC، config parser، desktop entry parser.
- [ ] **QA-04** (P1) رفع flakyها (`-count=20` در nightly).

### 15.2 Integration/E2E (روی بنیان `tests/e2e/run.sh`)
- [ ] **QA-05** (P0) تست‌های جدید e2e: daemon+Sway اتصال/reconnect، `nextctl status` واقعی، socket permission، config خراب، لاگین کامل DM (VM GUI)، logout، crash daemon/shell/lock، چندمانیتوره/scale، نبود optional deps، upgrade/uninstall، نصب `.deb`.
- [ ] **QA-06** (P1) اجرای e2e در CI (runner با `/dev/kvm` یا headless-only بدون VM برای subset) + آپلود artifactها (`tests/vm/.state/artifacts`).
- [ ] **QA-07** (P1) تست طولانی (soak ≥ 8h) با اندازه‌گیری RSS/FD/goroutine.
- [ ] **QA-08** (P1) ماتریس VM: Debian 13 و Ubuntu 24.04 (ایمیج دوم در `vm.sh`).

### 15.3 Manual QA
- [ ] **QA-10** (P0) `docs/development/qa-checklist.md`: keyboard/mouse/touchpad/چندمانیتوره، sleep/resume، قطع/وصل شبکه، تغییر دستگاه صدا، برنامه‌های GTK/Qt/Electron، notification انبوه، recovery از config/daemon/UI خراب.
- [ ] **QA-11** (P0) حداقل یک چرخهٔ استفادهٔ روزمره روی سخت‌افزار واقعی (نتیجه در `docs/release/rc-report.md`).

---

## 16. Performance و Reliability

- [ ] **PERF-01** (P1) Baseline و بودجه (ثبت در `docs/release/performance.md`): login→desktop آماده ≤ 5s، RAM idle daemon ≤ 50 MiB، shell ≤ 250 MiB، CPU idle < 2%.
- [ ] **PERF-02** (P1) اندازه‌گیری startup هر مولفه (trace اوقات در log).
- [ ] **PERF-03** (P1) Debounce/throttle eventهای پرتکرار (`window::title`, `input`) با حفظ eventهای ضروری.
- [ ] **PERF-04** (P1) سقف حافظه برای صف‌ها/cache (⛓ CORE-21).
- [ ] **PERF-05** (P1) تست disk-full، EACCES، process hang، سوکت پر.
- [ ] **PERF-06** (P1) بنچمارک `go test -bench` برای event bus/state/IPC parser و ثبت baseline در CI (هشدار regression).

---

## 17. مستندات

- [ ] **DOC-01** (P0) README: وضعیت واقعی، دستورات نصب per-distro (Ubuntu 24.04 / Debian 13)، لیست وابستگی کامل (شامل `foot`, `jq`, `rsync`) هم‌خوان با `guest-provision.sh`، build/run/test، لینک KVM lab.
- [ ] **DOC-02** (P0) `docs/user-guide/`: نصب، انتخاب سشن از DM، اولین اجرا، کلیدهای میانبر، config.
- [ ] **DOC-03** (P0) `docs/troubleshooting.md` + `docs/recovery.md` (config خراب، daemon نمی‌آید، UI سیاه، بازگشت به Sway خام).
- [ ] **DOC-04** (P1) `docs/api/` و `docs/config/` (⛓ API-10, CFG-11).
- [ ] **DOC-05** (P1) Backup/restore/upgrade/uninstall.
- [ ] **DOC-06** (P1) ADRها (`docs/architecture/adr/`) برای تصمیم‌های RPC، plugin، storage، session.
- [ ] **DOC-07** (P1) به‌روزرسانی `OVERVIEW.md` با دیاگرام واقعی و اصل «Core مستقل از UI».
- [ ] **DOC-08** (P1) Screenshotهای واقعی UI (از `vm.sh screenshot`) و Known Issues.
- [ ] **DOC-09** (P2) راهنمای plugin فقط در صورت فعال شدن PLG.

---

## 18. CI/CD و Release Engineering

- [ ] **CI-01** (P0) CI فعلی تقویت شود (⛓ FIX-12): ماتریس، cache، pin، `govulncheck`، `qmllint` واقعی، shellcheck.
- [ ] **CI-02** (P1) Job build روی Ubuntu 24.04 و Debian 13 (container) و بسته‌سازی `.deb` به‌عنوان artifact.
- [ ] **CI-03** (P1) Job e2e/VM (self-hosted یا nightly) (⛓ QA-06).
- [ ] **CI-04** (P0) Workflow release روی tag `vX.Y.Z`: build، تست، `.deb`، checksum، SBOM، امضا، GitHub Release با changelog.
- [ ] **CI-05** (P1) Dependabot/Renovate برای Go modules و Actions.
- [ ] **CI-06** (P1) بررسی Conventional Commits یا تولید changelog خودکار.
- [ ] **CI-07** (P0) RC (`v1.0.0-rc.N`) پیش از نهایی؛ مستند rollback/نصب نسخهٔ قبل.
- [ ] **CI-08** (P1) Branch protection: PR الزامی، CI سبز، حداقل یک review.

---

## 19. Release Gate --- چک‌لیست نهایی

> همهٔ موارد باید همزمان برقرار باشند. هر مورد به تست خودکار یا QA ثبت‌شده لینک شود.

- [ ] نصب `.deb` روی سیستم تمیز Ubuntu 24.04 **و** Debian 13 موفق (QA-05/PKG-14)
- [ ] ورود از DM به سشن i3_next؛ Sway+daemon+Quickshell خودکار (SES-09)
- [ ] پنل و launcher قابل استفاده (UI-10/11)
- [ ] مدیریت پنجره/workspace و چندمانیتوره (SWAY-20..22)
- [ ] Notificationها (SVC-03/UI-13)
- [ ] گزارش درست صدا/شبکه/باتری (SVC-04..06)
- [ ] lock/logout/reboot/shutdown مسیر امن؛ lock fail-closed (SVC-07, SES-05)
- [ ] قطع backend ⟵ UI نمی‌میرد و reconnect می‌شود (UI-01, API-09)
- [ ] config خراب ⟵ بدون از دست رفتن نشست و restart-loop (CFG-07)
- [ ] uninstall/upgrade داده کاربر را حذف نمی‌کنند (PKG-11)
- [ ] CI سبز؛ هیچ باگ blocker/critical باز نیست
- [ ] `govulncheck` و dependency scan بدون یافتهٔ بحرانی (SEC-04)
- [ ] مستندات نصب/عیب‌یابی/recovery منتشر شده (DOC-01..03)
- [ ] نسخه/commit/تاریخ/changelog در artifactها (PKG-02)
- [ ] RC روی سخت‌افزار واقعی تست و ثبت شده (QA-11)
- [ ] Known Issues و مسیر گزارش باگ منتشر شده (DOC-08, SEC-12)

---

## 20. Milestoneها با معیار خروج

| # | Milestone | بخش‌ها | معیار خروج (قابل‌اندازه‌گیری) |
|---|---|---|---|
| M0 | Scope & Quick wins | 0، 1 (`FIX-*`) | تمام `FIX-*` با اولویت P0 بسته؛ `make`، `make check`، `make vm-test` سبز؛ نسخه واحد |
| M1 | Core runtime | 2، 3، 12 | daemon با سرویس‌های fake: ترتیب/rollback/shutdown/supervisor/config تست‌شده؛ `nextctl status` واقعی |
| M2 | Sway & API | 4، 5 | e2e: daemon ⟷ Sway headless، reconnect، JSON-RPC با version+peercred، contract tests |
| M3 | First usable shell | 6، 7 (Must) | panel+launcher+session menu روی VM؛ چندمانیتوره؛ reconnect UI |
| M4 | Session | 8، 13 | login از DM در VM بدون دخالت؛ logout تمیز؛ recovery مستند |
| M5 | Persistence & Security | 10، 11 | threat model؛ permissionها؛ fuzz؛ lock fail-closed؛ `govulncheck` تمیز |
| M6 | Packaging & Docs | 14، 17 | `.deb` نصب/ارتقا/حذف روی هر دو توزیع؛ مستندات کامل |
| M7 | QA & Hardening | 15، 16 | soak ۸ ساعته بدون نشت؛ بودجهٔ performance؛ QA دستی |
| M8 | RC & Release | 18، 19 | `v1.0.0-rc.N` ⟵ Release Gate کامل ⟵ `v1.0.0` |

### ریسک‌های شناخته‌شده (پیگیری شوند)
| ریسک | اثر | کاهش |
|---|---|---|
| Quickshell در Ubuntu 24.04/Debian 13 موجود نیست یا قدیمی است | بلاکر نصب Tier-1 | تصمیم زودهنگام: PPA/backport/vendoring؛ تست در VM در M3 |
| نسخهٔ Go `1.27.3` ممکن است روی توزیع‌ها/CI در دسترس نباشد | بلاکر build | FIX-05 در M0 |
| Lock screen ناامن/نمایشی | ریسک امنیتی | فقط swaylock/ext-session-lock + تست fail-closed |
| دامنهٔ UI/سرویس‌ها بزرگ است | تأخیر | P1/P2 را در M3 دوباره اولویت‌بندی کن؛ سرویس‌های Must فقط بخش 7.2 با P0/P1 |
| نبود runner با KVM در CI | e2e فقط محلی | subset headless بدون VM در CI + nightly خودمیزبان |

---

## ثبت پیشرفت انتشار

| مورد | مقدار |
|---|---|
| Target version | `1.0.0` |
| Current status | Planning → M0 (quick wins) |
| Release owner | TBD |
| Target distributions | Ubuntu LTS 24.04, Debian Stable 13 (amd64) |
| Target release date | TBD |
| Latest release candidate | TBD |
| Known blockers | FIX-05 (نسخهٔ Go)، Quickshell روی Tier-1، تصمیم PLG-00/STO-00 |
