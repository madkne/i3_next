# i3_next --- Production v1 Release TODO

> هدف این سند، تبدیل وضعیت فعلی مخزن به اولین نسخه قابل نصب، قابل
> استفاده و قابل پشتیبانی از i3_next است.
>
> مبنای تهیه: snapshot فعلی مخزن، `README.md` و
> `docs/architecture/OVERVIEW.md`. وضعیت فعلی کد بسیار ابتدایی است:
> bootstrap فقط تا زمان لغو context منتظر می‌ماند، فایل‌های QML و config
> عمدتاً خالی‌اند و بیشتر قابلیت‌های معرفی‌شده در README هنوز پیاده‌سازی
> نشده‌اند.
>
> **قاعده تکمیل:** هر مورد فقط زمانی تیک بخورد که پیاده‌سازی، تست مرتبط و
> مستندات لازم آن انجام شده باشد.

## 0. تعریف محدوده و معیارهای انتشار

-   [x] محدوده دقیق v1 را تثبیت کن؛ قابلیت‌های خارج از این محدوده به
    backlog نسخه‌های بعد منتقل شوند.
-   [x] توزیع‌های هدف را مشخص کن (پیشنهاد اولیه: Ubuntu LTS و Debian
    Stable؛ سایر توزیع‌ها best-effort).
-   [x] معماری سخت‌افزار هدف را مشخص کن (حداقل x86_64؛ وضعیت ARM64 مستند
    شود).
-   [x] نسخه‌های پشتیبانی‌شده Sway، Wayland، Quickshell، Qt و Go را مشخص و
    در یک ماتریس سازگاری ثبت کن.
-   [x] حداقل سخت‌افزار، وابستگی‌های اجباری و اختیاری و فضای دیسک موردنیاز
    را مشخص کن.
-   [x] تصمیم بگیر v1 یک Desktop Session کامل مبتنی بر Sway ارائه می‌کند
    یا صرفاً shell قابل اجرا روی نشست موجود؛ پیشنهاد: نشست کامل Sway.
-   [x] فهرست قابلیت‌های Must-have نسخه اول را تثبیت کن.
-   [x] قابلیت‌های خارج از v1 را صریحاً ثبت کن: اجرای پلاگین شخص ثالث با
    sandbox قوی، همگام‌سازی ابری تنظیمات، چندکاربره/چندنشستی پیشرفته و
    امکاناتی که هنوز طراحی نشده‌اند.
-   [x] برای هر قابلیت v1 معیار پذیرش قابل آزمایش تعریف کن.
-   [x] سیاست نسخه‌بندی، سازگاری تنظیمات و پشتیبانی از ارتقای نسخه را
    تعیین کن.

### معیارهای خروج از این بخش

-   [x] سند `docs/RELEASE_SCOPE.md` شامل محدوده، پلتفرم‌ها، وابستگی‌ها و
    معیارهای پذیرش منتشر شده باشد.
-   [x] هیچ قابلیت README به‌عنوان قابلیت موجود معرفی نشده باشد مگر اینکه
    واقعاً پیاده‌سازی و تست شده باشد.

## 1. تکمیل ساختار پروژه و کیفیت پایه

-   [x] ساختار واقعی پروژه را با معماری README هماهنگ کن؛ پوشه‌های ضروری
    را ایجاد و موارد غیرضروری را حذف یا به آینده موکول کن.
-   [x] پکیج‌های Go را بر اساس مسئولیت تفکیک کن و وابستگی UI به core را
    ممنوع کن.
-   [x] قراردادهای مشترک، مدل‌های داده و خطاهای دامنه را تعریف کن.
-   [x] نام‌گذاری پروژه، module path، نام باینری‌ها و مسیرهای runtime را
    یکدست کن.
-   [x] نسخه Go موردنیاز را با نسخه پایدار و در دسترس toolchain هدف
    تطبیق بده؛ مقدار فعلی `go 1.27.3` را پیش از انتشار اعتبارسنجی کن.
-   [x] `go.mod` و dependencyها را تثبیت و dependencyهای بلااستفاده را
    حذف کن.
-   [x] فایل‌های خالی و TODOهای مبهم را یا پیاده‌سازی کن یا با issue مشخص
    و محدوده نسخه آینده جایگزین کن.
-   [x] `.gitignore` را برای خروجی‌های build، فایل‌های محلی، cache و
    داده‌های runtime کامل کن.
-   [x] `LICENSE` را با مجوز انتخاب‌شده تکمیل کن.
-   [x] فایل‌های `CONTRIBUTING.md` و `SECURITY.md` را اضافه کن.
-   [x] قالب‌بندی، lint و بررسی ایستای کد را در فرایند توسعه اجباری کن.

## 2. Desktop Core و چرخه عمر برنامه

### 2.1 Bootstrap و lifecycle

-   [ ] bootstrap واقعی را پیاده‌سازی کن؛ `App.Run` صرفاً منتظر
    `ctx.Done()` نباشد.
-   [ ] ترتیب راه‌اندازی اجزا را مشخص و deterministic کن.
-   [ ] برای هر سرویس قرارداد `Start`, `Stop`, `Health` و در صورت نیاز
    `Reload` تعریف کن.
-   [ ] راه‌اندازی مرحله‌ای و rollback در صورت شکست یکی از اجزای ضروری را
    پیاده‌سازی کن.
-   [ ] shutdown ترتیبی با timeout و آزادسازی منابع را پیاده‌سازی کن.
-   [ ] رفتار SIGTERM، SIGINT، crash و خروج غیرمنتظره dependencyها را
    تست کن.
-   [ ] از اجرای هم‌زمان چند نمونه daemon برای یک نشست جلوگیری کن.
-   [ ] وضعیت daemon و علت آخرین شکست را قابل مشاهده کن.

### 2.2 Process supervisor

-   [ ] supervisor برای اجرای processهای خارجی ایجاد کن.
-   [ ] مدیریت stdout/stderr، exit code، signal و timeout را پیاده‌سازی
    کن.
-   [ ] restart policy محدود و قابل تنظیم با backoff داشته باش.
-   [ ] از restart loop جلوگیری و وضعیت degraded ثبت کن.
-   [ ] child processها هنگام خروج daemon به‌درستی متوقف شوند.
-   [ ] environment هر process را کنترل و متغیرهای حساس را در log افشا
    نکن.
-   [ ] مسیر executableها را اعتبارسنجی کن و از shell interpolation
    ناامن پرهیز کن.

### 2.3 Event bus و state

-   [ ] event bus داخلی با قرارداد type-safe تعریف کن.
-   [ ] subscribe/unsubscribe و لغو subscription هنگام shutdown را
    پیاده‌سازی کن.
-   [ ] رفتار backpressure، صف پر، ترتیب رویداد و خطای handler را مشخص
    کن.
-   [ ] eventهای اصلی مانند `SessionStarted`, `SwayConnected`,
    `WorkspaceChanged`, `OutputChanged`, `ConfigChanged`,
    `ServiceStateChanged` و `ShutdownRequested` را تعریف کن.
-   [ ] state runtime را از config پایدار جدا نگه دار.
-   [ ] دسترسی هم‌زمان به state را race-safe کن.
-   [ ] تست بار و تست race برای event bus و state اضافه کن.

## 3. Configuration Management

-   [ ] schema نسخه‌دار برای `desktop.yaml`, `services.yaml`,
    `plugins.yaml` و `shortcuts.yaml` تعریف کن.
-   [ ] مقادیر پیش‌فرض معتبر برای تمام فایل‌های فعلی ایجاد کن؛ فایل‌های
    خالی باقی نمانند.
-   [ ] مسیر config کاربر را مطابق XDG Base Directory تعیین کن.
-   [ ] مسیر config سیستمی و ترتیب اولویت overrideها را تعریف کن.
-   [ ] در اولین اجرا config پیش‌فرض را بدون overwrite کردن تنظیمات کاربر
    نصب کن.
-   [ ] اعتبارسنجی schema و پیام خطای قابل فهم ارائه کن.
-   [ ] migration تنظیمات بین نسخه‌ها را طراحی و پیاده‌سازی کن.
-   [ ] reload امن تنظیمات را پشتیبانی کن؛ تغییرات غیرقابل reload باید
    صریحاً گزارش شوند.
-   [ ] نوشتن config را atomic انجام بده و از خراب شدن فایل در قطع
    برق/فرایند جلوگیری کن.
-   [ ] امکان backup و restore تنظیمات را فراهم کن.
-   [ ] تنظیمات نامعتبر نباید باعث crash یا ورود به restart loop شوند.
-   [ ] مستندات کامل schema و نمونه config اضافه کن.

## 4. Sway Integration

### 4.1 IPC client و adapter

-   [ ] Sway IPC client مستقل از سرویس‌های دامنه ایجاد کن.
-   [ ] اتصال به socket را از `SWAYSOCK` تشخیص بده و نبود یا نامعتبر
    بودن آن را مدیریت کن.
-   [ ] handshake، framing، serialization و parsing پاسخ‌های IPC را
    پیاده‌سازی کن.
-   [ ] command، query و subscribe به eventها را پشتیبانی کن.
-   [ ] reconnect با backoff و تشخیص قطع اتصال را پیاده‌سازی کن.
-   [ ] timeout، لغو context و خطاهای IPC را مدیریت کن.
-   [ ] مدل‌های typed برای outputs، workspaces، tree، inputs و windowها
    تعریف کن.
-   [ ] commandها را در adapter متمرکز کن؛ سرویس‌های بالادستی مستقیماً
    socket را دست‌کاری نکنند.
-   [ ] نسخه و قابلیت‌های Sway را در زمان اجرا تشخیص بده.
-   [ ] تفاوت نسخه‌های پشتیبانی‌شده را مستند و در صورت نیاز feature
    detection انجام بده.
-   [ ] تست unit با fixtureهای واقعی IPC و تست integration با Sway اجرا
    کن.

### 4.2 مدیریت پنجره و workspace

-   [ ] دریافت وضعیت پنجره‌ها و workspaceها را پیاده‌سازی کن.
-   [ ] رویدادهای window، workspace، output و mode را دریافت و به event
    bus تبدیل کن.
-   [ ] عملیات پایه focus، move، close، floating و layout را از API
    داخلی ارائه کن.
-   [ ] رفتار چند monitor و جابه‌جایی workspace بین خروجی‌ها را تست کن.
-   [ ] قواعد window matching را با schema مشخص پیاده‌سازی کن.
-   [ ] تنظیمات Sway را از templateهای کنترل‌شده تولید کن.
-   [ ] اعتبارسنجی config تولیدشده با `sway --validate` یا روش سازگار با
    نسخه هدف را انجام بده.
-   [ ] config سفارشی کاربر را بدون backup یا رضایت کاربر overwrite نکن.
-   [ ] کلیدهای پیش‌فرض و راه بازگشت به تنظیمات اصلی را مستند کن.

## 5. API بین Core و Shell

-   [ ] transport داخلی v1 را نهایی کن؛ Unix Domain Socket با JSON-RPC
    گزینه پیشنهادی است.
-   [ ] قرارداد API و eventها را در `docs/api/` مستند کن.
-   [ ] نسخه API و روش تشخیص ناسازگاری client/server را تعریف کن.
-   [ ] socket را فقط با permission مناسب در runtime directory کاربر
    ایجاد کن.
-   [ ] مالکیت socket، cleanup فایل stale و جلوگیری از اتصال نشست دیگر
    را پیاده‌سازی کن.
-   [ ] احراز هویت محلی/کنترل peer credentials را متناسب با مدل تهدید
    تعریف کن.
-   [ ] درخواست‌ها، پاسخ‌ها، خطاها و notificationها را استاندارد کن.
-   [ ] محدودیت اندازه پیام، timeout و تعداد درخواست‌های هم‌زمان را اعمال
    کن.
-   [ ] event stream را با reconnect و بازیابی snapshot وضعیت پشتیبانی
    کن.
-   [ ] APIهای status، config، service، workspace، output، notification
    و launcher را تعریف کن.
-   [ ] خطاهای API برای UI قابل نمایش و قابل تشخیص باشند.
-   [ ] contract test برای Go و QML اضافه کن.

## 6. Quickshell Desktop Shell

### 6.1 زیرساخت UI

-   [ ] حداقل نسخه Quickshell و Qt/QML موردنیاز را تعیین و بررسی کن.
-   [ ] ساختار QML را به ماژول‌ها و componentهای کوچک تقسیم کن.
-   [ ] `qmldir` و import pathها را کامل کن.
-   [ ] `Main.qml`, `Shell.qml` و `Session.qml` را به یک جریان راه‌اندازی
    واقعی متصل کن.
-   [ ] اتصال Backend را با API تعریف‌شده پیاده‌سازی کن.
-   [ ] reconnect خودکار به daemon و نمایش وضعیت اتصال را اضافه کن.
-   [ ] eventهای backend را به modelهای QML تبدیل کن.
-   [ ] خطاهای QML و backend را بدون crash کل shell مدیریت کن.
-   [ ] logging و debug mode مخصوص UI اضافه کن.
-   [ ] طراحی responsive برای رزولوشن‌ها و scale factorهای متفاوت انجام
    بده.
-   [ ] پشتیبانی از چند monitor و lifecycle جداگانه panelها را پیاده‌سازی
    کن.
-   [ ] دسترس‌پذیری پایه، focus keyboard، اندازه متن و contrast را رعایت
    کن.
-   [ ] theme tokenها و رنگ‌ها را از componentها جدا کن.

### 6.2 قابلیت‌های ضروری Shell

-   [ ] پنل اصلی با ساعت، workspaceها، وضعیت شبکه، صدا و باتری (در صورت
    وجود) بساز.
-   [ ] task/window switcher یا نمایش پنجره‌های باز را اضافه کن.
-   [ ] application launcher با جست‌وجوی برنامه‌های نصب‌شده و اجرای امن
    desktop entryها بساز.
-   [ ] notification popup و Notification Center بساز.
-   [ ] Control Center برای تنظیمات سریع صدا، روشنایی، شبکه و Bluetooth
    (در صورت پشتیبانی سخت‌افزار) بساز.
-   [ ] OSD برای volume، brightness و تغییر workspace بساز.
-   [ ] نمایش وضعیت session و عملیات logout/restart/shutdown را اضافه
    کن.
-   [ ] تنظیمات ظاهری و مدیریت theme پایه را ارائه کن.
-   [ ] وضعیت empty، loading، error و disconnected برای تمام صفحه‌ها
    طراحی شود.
-   [ ] رفتار کلیک، keyboard navigation و shortcutهای UI را تست کن.

## 7. قابلیت‌های Desktop و سرویس‌های سیستم

> برای اولین release، قابلیت‌های ضروری را از قابلیت‌های تکمیلی جدا نگه
> دار. هر سرویس باید مستقل باشد و در صورت نبود dependency اختیاری، کل
> Desktop از کار نیفتد.

### 7.1 سرویس‌های Must-have

-   [ ] Session manager: شروع، پایان و بازیابی نشست.
-   [ ] Wallpaper manager: انتخاب تصویر، حالت نمایش و پشتیبانی چند
    monitor.
-   [ ] Notification service: دریافت notification از D-Bus و نمایش در
    Shell.
-   [ ] Audio service: مشاهده و تغییر خروجی/ورودی و volume از طریق
    PipeWire/WirePlumber.
-   [ ] Network status: نمایش اتصال و وضعیت شبکه؛ مدیریت کامل اتصال
    می‌تواند به NetworkManager وابسته باشد.
-   [ ] Power service: وضعیت باتری، شارژ و عملیات power با رعایت مجوزهای
    سیستم.
-   [ ] Idle manager: اجرای lock/suspend policy با تنظیمات قابل کنترل.
-   [ ] Lock screen: قفل نشست با ابزار امن و یکپارچگی با Sway.
-   [ ] Screenshot service: گرفتن screenshot از خروجی یا ناحیه انتخابی.
-   [ ] Clipboard integration: حداقل مشاهده/مدیریت clipboard در محدوده‌ای
    که ابزارهای Wayland پشتیبانی می‌کنند.
-   [ ] Display service: نمایش خروجی‌ها و تنظیم رزولوشن/مقیاس با
    اعتبارسنجی.
-   [ ] Media controls: کنترل playback از طریق MPRIS، در صورت وجود
    player سازگار.

### 7.2 قابلیت‌های اختیاری یا قابل تعویق

-   [ ] Bluetooth manager کامل.
-   [ ] File search و command palette پیشرفته.
-   [ ] Clipboard history پایدار و جست‌وجوپذیر.
-   [ ] Layout manager پیشرفته.
-   [ ] Color picker.
-   [ ] System monitor پیشرفته.
-   [ ] Workspace overview و انیمیشن‌های پیشرفته.
-   [ ] Power profiles و تنظیمات سازنده‌محور.

### 7.3 الزامات مشترک سرویس‌ها

-   [ ] هر سرویس interface مستقل و dependencyهای مشخص داشته باشد.
-   [ ] قابلیت enable/disable از config فراهم شود.
-   [ ] نبود dependency اختیاری به وضعیت degraded منجر شود، نه crash کل
    daemon.
-   [ ] تغییرات وضعیت از event bus منتشر شوند.
-   [ ] عملیات طولانی cancellable و دارای timeout باشند.
-   [ ] خطاهای دسترسی D-Bus و ابزارهای سیستم به شکل قابل فهم گزارش شوند.
-   [ ] هیچ دستور privileged بدون طراحی مجوز و بررسی امنیت اجرا نشود.
-   [ ] تست unit و integration برای هر سرویس اضافه شود.

## 8. D-Bus و System Integration

-   [ ] مشخص کن کدام APIها از session bus و کدام از system bus استفاده
    می‌کنند.
-   [ ] clientهای D-Bus با reconnect و timeout بساز.
-   [ ] interfaceهای لازم برای notification، power، NetworkManager،
    MPRIS و سایر سرویس‌ها را فهرست کن.
-   [ ] در صورت نیاز، D-Bus service file و policyهای لازم را اضافه کن.
-   [ ] از اجرای commandهای shell با داده ورودی کاربر اجتناب کن.
-   [ ] نبود daemonهای اختیاری را مدیریت کن.
-   [ ] تعامل با logind و مجوزهای suspend/reboot/shutdown را بررسی کن.
-   [ ] تست روی نشست واقعی D-Bus و نشست ناقص/بدون سرویس‌های اختیاری انجام
    بده.

## 9. Plugin System

-   [ ] تصمیم بگیر plugin system در v1 واقعاً لازم است یا به نسخه بعد
    منتقل شود.
-   [ ] اگر در v1 باقی می‌ماند، manifest نسخه‌دار و schema اعتبارسنجی‌شده
    تعریف کن.
-   [ ] lifecycle و APIهای مجاز plugin را محدود کن.
-   [ ] permission model و اصل حداقل دسترسی را پیاده‌سازی کن.
-   [ ] plugin ناشناس یا ناسازگار به‌صورت پیش‌فرض اجرا نشود.
-   [ ] امضای بسته یا روش اعتماد به منبع plugin را طراحی کن.
-   [ ] resource limit، timeout و مدیریت crash را تعریف کن.
-   [ ] مرز امنیتی sandbox را واقع‌بینانه مستند کن؛ اجرای process جدا
    به‌تنهایی sandbox امن محسوب نمی‌شود.
-   [ ] نمونه plugin رسمی و مستندات SDK بساز.
-   [ ] تست سازگاری نسخه API و تست رفتار plugin مخرب/خراب اضافه کن.
-   [ ] اگر sandbox قابل اتکا برای v1 آماده نیست، اجرای plugin شخص ثالث
    را غیرفعال و فقط built-inها را مجاز کن.

## 10. Storage و داده پایدار

-   [ ] مشخص کن کدام داده‌ها واقعاً نیاز به persistence دارند.
-   [ ] مسیر پایگاه داده را مطابق XDG تعیین کن.
-   [ ] SQLite را فقط برای داده‌های لازم استفاده کن؛ config متنی همچنان
    قابل ویرایش بماند.
-   [ ] schema و migration نسخه‌دار اضافه کن.
-   [ ] عملیات DB را از UI و سرویس‌ها پشت repository جدا کن.
-   [ ] backup و restore داده‌های کاربر را پیاده‌سازی کن.
-   [ ] رفتار خرابی DB و migration ناموفق را تست کن.
-   [ ] داده‌های حساس را ذخیره نکن مگر ضرورت و سیاست حفاظت روشن داشته
    باشند.
-   [ ] retention و پاک‌سازی cache/history را مشخص کن.

## 11. Security و Privacy

-   [ ] threat model نسخه اول را در `docs/SECURITY.md` ثبت کن.
-   [ ] تمام فایل‌ها و socketهای runtime با permission حداقلی ساخته شوند.
-   [ ] از path traversal و symlink attack در مسیرهای config، plugin و
    asset جلوگیری کن.
-   [ ] اجرای command و process با آرگومان‌های امن و بدون shell injection
    انجام شود.
-   [ ] ورودی‌های IPC، D-Bus، config و plugin اعتبارسنجی شوند.
-   [ ] logها نباید اطلاعات حساس کاربر را ثبت کنند.
-   [ ] اجرای عملیات privileged به روش استاندارد سیستم‌عامل و با حداقل
    دسترسی باشد.
-   [ ] lock screen از ابزار/مکانیزم امن و آزموده‌شده استفاده کند؛ lock
    screen نمایشی یا غیرقابل اعتماد پذیرفته نیست.
-   [ ] رفتار هنگام crash شدن shell یا lock component بررسی شود تا نشست
    ناخواسته باز نماند.
-   [ ] dependencyها از نظر آسیب‌پذیری بررسی شوند.
-   [ ] مجوزها و منبع dependencyهای third-party ثبت شوند.
-   [ ] سیاست جمع‌آوری telemetry (پیشنهاد: بدون telemetry پیش‌فرض) مشخص
    شود.

## 12. Logging، Diagnostics و Observability

-   [ ] logging ساختاریافته با سطح‌های debug/info/warn/error پیاده‌سازی
    کن.
-   [ ] خروجی log به journal یا مسیر استاندارد کاربر هدایت شود.
-   [ ] log rotation و محدودیت حجم را در نظر بگیر.
-   [ ] نام component، نسخه، commit و شناسه نشست در logها درج شود.
-   [ ] خطاهای قابل اقدام از خطاهای عادی جدا شوند.
-   [ ] `i3-next-doctor` بررسی‌های ضروری را انجام دهد: Sway، Quickshell،
    Wayland، D-Bus، ابزارهای موردنیاز، permissionها، config و API
    socket.
-   [ ] doctor برای هر بررسی وضعیت OK/WARN/FAIL و راهکار اصلاح ارائه
    کند.
-   [ ] `i3-nextctl status` وضعیت daemon و سرویس‌ها را از API واقعی
    بخواند.
-   [ ] `i3-nextctl` فرمان‌های ضروری مثل start/status/stop/restart/config
    validate را ارائه کند؛ فقط فرمان‌هایی که backend امن و واقعی دارند.
-   [ ] خروجی machine-readable مانند JSON برای ابزارهای عیب‌یابی فراهم
    کن.
-   [ ] گزارش diagnostics قبل از اشتراک‌گذاری، مسیرها و اطلاعات حساس را
    حذف یا redact کند.

## 13. Session، Login و مدیریت نشست

-   [ ] فایل desktop session برای display managerهای هدف بساز.
-   [ ] ترتیب اجرای D-Bus، Sway، daemon و Quickshell را مشخص کن.
-   [ ] environment لازم مانند `XDG_CURRENT_DESKTOP`,
    `XDG_SESSION_TYPE`, `WAYLAND_DISPLAY` و `SWAYSOCK` را بررسی و تنظیم
    کن.
-   [ ] از اجرای تکراری daemon یا shell هنگام login جلوگیری کن.
-   [ ] startup failure باید پیام خطا و مسیر recovery داشته باشد.
-   [ ] logout باید سرویس‌ها و processهای متعلق به نشست را تمیز متوقف
    کند.
-   [ ] restart shell بدون logout کامل امکان‌پذیر باشد.
-   [ ] رفتار suspend/resume و lock/unlock را تست کن.
-   [ ] ورود از display manager و اجرای دستی برای توسعه هر دو مستند
    شوند.

## 14. Packaging و نصب

### 14.1 Build و release artifacts

-   [ ] Makefile را برای build قابل تکرار و قابل پاک‌سازی کامل کن.
-   [ ] `build_date` و `commit` را در تمام باینری‌ها به شکل یکسان inject
    کن.
-   [ ] نسخه release را از یک منبع واحد تولید کن؛ مقدارهای ثابت تکراری
    در `info.go` حذف شوند.
-   [ ] build تمام باینری‌ها را در CI انجام بده.
-   [ ] خروجی‌های debug و release را تفکیک کن.
-   [ ] dependencyهای runtime را شناسایی و در metadata بسته ثبت کن.
-   [ ] SBOM برای release تولید کن.
-   [ ] checksum و امضای artifactها را منتشر کن.

### 14.2 بسته‌های توزیع

-   [ ] بسته Debian/Ubuntu با dependency و scriptهای درست بساز.
-   [ ] بسته Arch Linux یا PKGBUILD رسمی تهیه کن، اگر در محدوده پشتیبانی
    v1 است.
-   [ ] فایل systemd user service در صورت نیاز اضافه کن.
-   [ ] فایل D-Bus service و policyهای لازم را بسته‌بندی کن.
-   [ ] فایل Wayland session برای display managerهای هدف نصب کن.
-   [ ] فایل‌های config پیش‌فرض را در مسیر مناسب نصب کن.
-   [ ] فایل‌های QML، themeها، iconها و assetها در مسیر قابل کشف نصب
    شوند.
-   [ ] uninstall نباید config و داده شخصی کاربر را بدون رضایت حذف کند.
-   [ ] upgrade بسته باید config موجود را حفظ کند.
-   [ ] نصب و حذف را روی VM تمیز تست کن.
-   [ ] نصب آفلاین یا نصب از repository را برای v1 تعیین تکلیف کن.

### 14.3 Installer scripts

-   [ ] `scripts/dev-setup.sh` را idempotent و مخصوص محیط توسعه کن.
-   [ ] `scripts/build.sh` را با Makefile هماهنگ کن یا حذف کن تا منبع
    build دوگانه ایجاد نشود.
-   [ ] `scripts/install.sh` فقط روش پشتیبانی‌شده نصب را اجرا کند و قبل
    از تغییر سیستم پیش‌نیازها را بررسی کند.
-   [ ] `scripts/uninstall.sh` فقط فایل‌های متعلق به برنامه را حذف کند.
-   [ ] تمام scriptها با shellcheck بررسی شوند.
-   [ ] اجرای script با مسیر دارای فاصله و خطاهای میانی تست شود.
-   [ ] هیچ scriptی بدون توضیح، فایل‌های کاربر یا تنظیمات موجود را
    overwrite نکند.

## 15. Testing و تضمین کیفیت

### 15.1 Go tests

-   [ ] برای packageهای core، config، lifecycle، supervisor، API و
    adapter تست unit بنویس.
-   [ ] تست خطا و edge case برای تمام مسیرهای حیاتی اضافه کن.
-   [ ] `go test -race ./...` در CI اجرا شود.
-   [ ] `go vet ./...` و lint اجرا شوند.
-   [ ] coverage بخش‌های حیاتی اندازه‌گیری شود؛ coverage بالا جایگزین تست
    رفتاری نیست.
-   [ ] تست‌های flaky شناسایی و رفع شوند.

### 15.2 Integration و E2E

-   [ ] تست daemon با Sway واقعی در محیط headless یا nested انجام بده.
-   [ ] تست اتصال و reconnect بین daemon و Quickshell اضافه کن.
-   [ ] تست start/login تا نمایش کامل desktop را خودکار کن.
-   [ ] تست logout، shutdown، restart و crash componentها را اضافه کن.
-   [ ] تست چند monitor، رزولوشن و scale factor را اجرا کن.
-   [ ] تست نبودن optional dependencies را اجرا کن.
-   [ ] تست upgrade از نسخه قبلی و حفظ تنظیمات را اضافه کن.
-   [ ] تست نصب روی VM تمیز برای هر توزیع هدف داشته باش.
-   [ ] تست طولانی‌مدت session برای memory leak و افزایش مصرف منابع انجام
    بده.

### 15.3 Manual QA

-   [ ] چک‌لیست تست دستی برای keyboard، mouse، touchpad و چند monitor
    تهیه کن.
-   [ ] رفتار sleep/resume، network reconnect و audio device change را
    تست کن.
-   [ ] رفتار برنامه‌های GTK و Qt و desktop entryها را بررسی کن.
-   [ ] رفتار notificationهای پرتعداد و طولانی را بررسی کن.
-   [ ] بررسی کن shell یا daemon در حالت عادی CPU/RAM غیرمنطقی مصرف
    نکند.
-   [ ] مسیر recovery از config خراب، daemon crash و UI crash را آزمایش
    کن.
-   [ ] حداقل یک چرخه استفاده روزمره روی سخت‌افزار واقعی انجام بده.

## 16. Performance و Reliability

-   [ ] baseline زمان login تا آماده شدن desktop را اندازه‌گیری کن.
-   [ ] baseline مصرف RAM/CPU در حالت idle ثبت کن.
-   [ ] زمان startup هر component را اندازه‌گیری کن.
-   [ ] تعداد processهای اضافی و مصرف منابع UI را بررسی کن.
-   [ ] eventهای پرتکرار Sway را debounce/throttle کن، بدون از دست دادن
    eventهای ضروری.
-   [ ] reconnect و retryها را با backoff و سقف مناسب پیاده‌سازی کن.
-   [ ] صف‌ها و cacheها محدودیت حافظه داشته باشند.
-   [ ] رفتار در شرایط disk full، permission denied و process hang تست
    شود.
-   [ ] معیارهای عملکرد v1 را در سند release ثبت کن.

## 17. Documentation و تجربه توسعه‌دهنده

-   [ ] README را با وضعیت واقعی پروژه و قابلیت‌های پیاده‌سازی‌شده
    به‌روزرسانی کن.
-   [ ] دستورهای نصب وابستگی‌ها را برای هر توزیع هدف جداگانه بنویس.
-   [ ] راهنمای build، run، debug و اجرای تست را تکمیل کن.
-   [ ] راهنمای نصب و انتخاب i3_next از صفحه login بنویس.
-   [ ] راهنمای اولین اجرا و تنظیمات پایه را اضافه کن.
-   [ ] راهنمای کلیدهای میانبر پیش‌فرض را اضافه کن.
-   [ ] راهنمای config و schemaها را بنویس.
-   [ ] مستندات API و eventها را تکمیل کن.
-   [ ] راهنمای عیب‌یابی و استفاده از `i3-next-doctor` را اضافه کن.
-   [ ] راهنمای backup، restore، upgrade و uninstall بنویس.
-   [ ] مستندات توسعه plugin را فقط در صورت آماده بودن و پشتیبانی واقعی
    منتشر کن.
-   [ ] screenshot و نمونه‌های واقعی UI را به README اضافه کن.
-   [ ] محدودیت‌های شناخته‌شده و سخت‌افزار/توزیع‌های تست‌شده را اعلام کن.

## 18. CI/CD و Release Engineering

-   [ ] CI برای pull requestها راه‌اندازی کن.
-   [ ] pipeline شامل format check، lint، vet، unit test و build باشد.
-   [ ] build روی تمام پلتفرم‌های هدف انجام شود.
-   [ ] QML lint و اعتبارسنجی importها در CI اجرا شود.
-   [ ] shellcheck برای scriptها اجرا شود.
-   [ ] تست integration در runner مناسب یا VM انجام شود.
-   [ ] dependency vulnerability scan اضافه کن.
-   [ ] release workflow با tag نسخه اجرا شود.
-   [ ] changelog از تغییرات release تهیه شود.
-   [ ] artifactهای بسته‌بندی‌شده، checksum، امضا و SBOM منتشر شوند.
-   [ ] فرایند rollback یا نصب نسخه قبلی مستند شود.
-   [ ] release candidate پیش از نسخه نهایی منتشر و تست شود.

## 19. Release Gate --- چک‌لیست نهایی انتشار

نسخه v1 فقط زمانی آماده انتشار است که تمام موارد زیر برقرار باشند:

-   [ ] نصب روی حداقل یک سیستم تمیز از هر توزیع اصلی پشتیبانی‌شده موفق
    باشد.
-   [ ] کاربر بتواند از display manager وارد نشست i3_next شود.
-   [ ] Sway، daemon و Quickshell بدون دخالت دستی راه‌اندازی شوند.
-   [ ] پنل و launcher قابل استفاده باشند.
-   [ ] مدیریت پایه پنجره‌ها و workspaceها کار کند.
-   [ ] notificationها نمایش داده شوند.
-   [ ] صدا، شبکه و وضعیت باتری (در سخت‌افزار مربوط) به‌درستی گزارش شوند.
-   [ ] lock، logout، reboot و shutdown مسیر امن و قابل اتکا داشته
    باشند.
-   [ ] قطع اتصال backend باعث crash دائمی UI نشود و reconnect انجام
    شود.
-   [ ] config خراب باعث از دست رفتن نشست یا restart loop نشود.
-   [ ] uninstall و upgrade داده‌های شخصی را ناخواسته حذف نکنند.
-   [ ] تست‌های CI موفق باشند و باگ blocker یا critical باز وجود نداشته
    باشد.
-   [ ] هیچ آسیب‌پذیری بحرانی شناخته‌شده در dependencyها یا کد باقی نمانده
    باشد.
-   [ ] مستندات نصب، عیب‌یابی و recovery منتشر شده باشند.
-   [ ] نسخه، commit، تاریخ build و changelog در artifactها قابل تشخیص
    باشند.
-   [ ] Release Candidate روی سخت‌افزار واقعی تست و نتیجه ثبت شده باشد.
-   [ ] Known Issues و مسیر گزارش bug منتشر شده باشد.

## 20. ترتیب پیشنهادی اجرا

این ترتیب از ساخت زیرساخت به قابلیت قابل مشاهده و سپس سخت‌سازی و انتشار
می‌رود:

1.  **Milestone 0 --- Scope & baseline:** بخش‌های 0 و 1
2.  **Milestone 1 --- Core runtime:** بخش‌های 2، 3 و 12
3.  **Milestone 2 --- Sway & API:** بخش‌های 4 و 5
4.  **Milestone 3 --- First usable shell:** بخش 6 و قابلیت‌های ضروری بخش
    7
5.  **Milestone 4 --- Session integration:** بخش‌های 8 و 13
6.  **Milestone 5 --- Persistence & security:** بخش‌های 10 و 11
7.  **Milestone 6 --- Packaging & docs:** بخش‌های 14 و 17
8.  **Milestone 7 --- QA & hardening:** بخش‌های 15 و 16
9.  **Milestone 8 --- RC & production release:** بخش‌های 18 و 19

### اولویت پیشنهادی

-   **P0 --- Release blocker:** ورود به نشست، Sway IPC پایدار، daemon
    lifecycle، API امن و versioned، shell پایه، launcher، panel، config
    معتبر، lock/logout، نصب و uninstall امن، تست و diagnostics.
-   **P1 --- Expected v1:** notification center، audio/network/power
    status، wallpaper، screenshot، display basics، settings پایه،
    بسته‌بندی و مستندات کامل.
-   **P2 --- Defer if needed:** plugin شخص ثالث و sandbox، clipboard
    history پیشرفته، file search، color picker، layout manager پیشرفته،
    انیمیشن‌ها و ابزارهای تکمیلی.

------------------------------------------------------------------------

## ثبت پیشرفت انتشار

  مورد                       مقدار
  -------------------------- ----------
  Target version             `1.0.0`
  Current status             Planning
  Release owner              TBD
  Target distributions       Ubuntu LTS 24.04, Debian Stable 13 (amd64)
  Target release date        TBD
  Latest release candidate   TBD
  Known blockers             TBD
