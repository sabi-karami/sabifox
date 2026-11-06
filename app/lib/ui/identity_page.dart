import 'dart:async';
import 'dart:io';
import 'dart:ui' show PlatformDispatcher;

import 'package:flutter/material.dart';

import '../core/core_controller.dart';
import '../platform.dart';

/// The online check of what a browser shows (the site's /check page).
const identityCheckUrl = 'https://sabifox.sahebkarami.ir/check';

/// The Identity Shield: what on this device still points at Iran while
/// the VPN hides the address (the time zone, the languages, a Persian
/// keyboard, Persian fonts), how traffic is kept inside the VPN, and the
/// time zone sync that follows the exit country (GET/PUT /api/identity).
class IdentityPage extends StatefulWidget {
  const IdentityPage({super.key, required this.core});

  final CoreController core;

  @override
  State<IdentityPage> createState() => _IdentityPageState();
}

enum _Level { ok, warn, bad, info }

class _Check {
  const _Check(this.level, this.icon, this.title, this.detail, [this.fix]);

  final _Level level;
  final IconData icon;
  final String title;
  final String detail;
  final String? fix;
}

class _IdentityPageState extends State<IdentityPage> {
  Map<String, Object?>? _report;
  String? _error;
  bool _busy = false;
  Timer? _timer;

  @override
  void initState() {
    super.initState();
    _load();
    // The sync may change the zone a few seconds after connecting.
    _timer = Timer.periodic(const Duration(seconds: 8), (_) => _load());
  }

  @override
  void dispose() {
    _timer?.cancel();
    super.dispose();
  }

  Future<void> _load() async {
    try {
      final r = await widget.core.call((api) => api.identity());
      if (!mounted) return;
      setState(() {
        _report = r;
        _error = null;
      });
    } catch (e) {
      if (!mounted) return;
      setState(() => _error = '$e');
    }
  }

  Future<void> _setSync(bool on) async {
    setState(() => _busy = true);
    try {
      final r = await widget.core.call((api) => api.setIdentitySync(on));
      if (!mounted) return;
      setState(() {
        _report = r;
        _error = null;
      });
    } catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('$e')));
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  Future<void> _setTrackers(bool on) async {
    setState(() => _busy = true);
    try {
      final r = await widget.core.call((api) => api.setBlockTrackers(on));
      if (!mounted) return;
      setState(() => _report = r);
    } catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('$e')));
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  Future<void> _launch(String id) async {
    try {
      await widget.core.call((api) => api.launchPrivateBrowser(id));
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text(
            _t('مرورگر خصوصی باز شد', 'The private browser started'),
          ),
        ),
      );
    } catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('$e')));
    }
  }

  bool get _fa => Localizations.localeOf(context).languageCode == 'fa';

  String _t(String fa, String en) => _fa ? fa : en;

  static Map<String, Object?> _map(Object? v) =>
      v is Map<String, Object?> ? v : const {};

  static String _str(Object? v) => v is String ? v : '';

  static int? _int(Object? v) => v is num ? v.toInt() : null;

  static String _offset(int minutes) {
    final sign = minutes < 0 ? '-' : '+';
    final m = minutes.abs();
    final h = (m ~/ 60).toString().padLeft(2, '0');
    final r = (m % 60).toString().padLeft(2, '0');
    return 'UTC$sign$h:$r';
  }

  List<_Check> _checks(Map<String, Object?> r) {
    final checks = <_Check>[];
    final connected = r['connected'] == true;
    final exitCountry = _str(r['exit_country']);
    final exitZone = _str(r['exit_zone']);
    final exitOffset = _int(r['exit_offset_minutes']);
    final sync = _map(r['sync']);
    final desktop = _map(r['desktop']);
    final leaks = _map(r['leaks']);
    final deviceOffset = DateTime.now().timeZoneOffset.inMinutes;
    final iranZone = r['system_iran'] == true || deviceOffset == 210;

    checks.add(
      connected && exitCountry.isNotEmpty
          ? _Check(
              _Level.ok,
              Icons.vpn_lock_rounded,
              _t('وی‌پی‌ان وصل است', 'VPN connected'),
              _t('خروجی: $exitCountry', 'Exit: $exitCountry'),
            )
          : _Check(
              _Level.bad,
              Icons.vpn_lock_rounded,
              _t('وی‌پی‌ان وصل نیست', 'VPN not connected'),
              _t(
                'سایت‌ها نشانی واقعی شما را می‌بینند.',
                'Sites see your real address.',
              ),
              _t('وصل شوید.', 'Connect.'),
            ),
    );

    // The time zone against the exit's.
    final zoneText = _t(
      'منطقه‌ی زمانی دستگاه: ${_offset(deviceOffset)}',
      'Device time zone: ${_offset(deviceOffset)}',
    );
    if (connected && exitOffset != null) {
      final match = exitOffset == deviceOffset;
      checks.add(
        _Check(
          match ? _Level.ok : (iranZone ? _Level.bad : _Level.warn),
          Icons.schedule_rounded,
          match
              ? _t('منطقه‌ی زمانی با کشور خروجی یکی است', 'Time zone matches the exit')
              : _t('منطقه‌ی زمانی با کشور خروجی فرق دارد', 'Time zone differs from the exit'),
          '$zoneText\n${_t('کشور خروجی', 'Exit')}: $exitZone (${_offset(exitOffset)})',
          match ? null : _zoneFix(sync, exitZone),
        ),
      );
    } else {
      checks.add(
        _Check(
          iranZone ? _Level.warn : _Level.info,
          Icons.schedule_rounded,
          iranZone
              ? _t('منطقه‌ی زمانی ایران است', 'The time zone is Iran\'s')
              : _t('منطقه‌ی زمانی', 'Time zone'),
          zoneText,
          iranZone
              ? _t(
                  'سایت‌ها منطقه‌ی زمانی را بدون اجازه می‌خوانند. پس از اتصال، همگام‌سازی پایین آن را با کشور خروجی یکی می‌کند.',
                  'Sites read the time zone without asking. Once connected, the sync below matches it to the exit.',
                )
              : null,
        ),
      );
    }

    // The languages browsers send (Accept-Language, navigator.languages).
    final locales = PlatformDispatcher.instance.locales;
    final langs = <String>{
      for (final l in locales) l.languageCode,
      for (final l in (desktop['languages'] as List<Object?>? ?? const []))
        _str(l).split('-').first.toLowerCase(),
    }..removeWhere((l) => l.isEmpty);
    final persianFirst =
        locales.isNotEmpty && locales.first.languageCode == 'fa';
    final persian = langs.contains('fa');
    checks.add(
      _Check(
        persianFirst ? _Level.bad : (persian ? _Level.warn : _Level.ok),
        Icons.translate_rounded,
        persian
            ? _t('زبان فارسی در فهرست زبان‌های سیستم است', 'Persian is in the system languages')
            : _t('زبان سیستم فارسی نیست', 'No Persian system language'),
        '${_t('زبان‌ها', 'Languages')}: ${langs.join('، ')}',
        persian
            ? _t(
                'مرورگرها زبان‌ها را برای هر سایت می‌فرستند. در تنظیمات مرورگر (Languages) انگلیسی را اول بگذارید و فارسی را بردارید؛ زبان رابط این برنامه جداست و اشکالی ندارد.',
                'Browsers send the languages to every site. In the browser\'s Languages settings, put English first and remove Persian.',
              )
            : null,
      ),
    );

    final keyboard = desktop['persian_keyboard'];
    if (keyboard is bool) {
      checks.add(
        _Check(
          keyboard ? _Level.warn : _Level.ok,
          Icons.keyboard_rounded,
          keyboard
              ? _t('صفحه‌کلید فارسی نصب است', 'A Persian keyboard is installed')
              : _t('صفحه‌کلید فارسی نیست', 'No Persian keyboard'),
          _t(
            'کروم و اج چیدمان صفحه‌کلید را به سایت‌ها نشان می‌دهند (navigator.keyboard).',
            'Chrome and Edge show the keyboard layout to sites (navigator.keyboard).',
          ),
          keyboard
              ? _t(
                  'هنگام وب‌گردی حساس، صفحه‌کلید فارسی را موقتاً حذف کنید یا از فایرفاکس استفاده کنید که این را نشان نمی‌دهد.',
                  'For sensitive browsing remove the Persian keyboard for a while, or use Firefox, which does not expose it.',
                )
              : null,
        ),
      );
    }

    final fonts = [
      for (final f in (desktop['persian_fonts'] as List<Object?>? ?? const []))
        _str(f),
    ];
    if (desktop.isNotEmpty) {
      checks.add(
        _Check(
          fonts.isEmpty ? _Level.ok : _Level.warn,
          Icons.font_download_rounded,
          fonts.isEmpty
              ? _t('فونت فارسی خاصی نصب نیست', 'No Persian fonts installed')
              : _t('${fonts.length} فونت فارسی نصب است', '${fonts.length} Persian fonts installed'),
          fonts.isEmpty
              ? _t('سایت‌ها با اندازه‌گیری متن، فونت‌های نصب‌شده را تشخیص می‌دهند.', 'Sites detect installed fonts by measuring text.')
              : '${fonts.take(8).join('، ')}${fonts.length > 8 ? ' …' : ''}',
          fonts.isEmpty
              ? null
              : _t(
                  'فونت‌هایی مثل وزیر، نازنین و ایران‌سنس تقریباً فقط در ایران نصب می‌شوند. حذفشان کنید، یا از فایرفاکس با privacy.resistFingerprinting یا Brave استفاده کنید که فونت‌ها را پنهان می‌کنند.',
                  'Fonts such as Vazir, Nazanin and IRANSans are installed almost only in Iran. Remove them, or use Firefox with privacy.resistFingerprinting, or Brave, which hide fonts.',
                ),
        ),
      );
    }

    // Traffic outside the VPN.
    final mode = _str(leaks['mode']);
    final tun = mode == 'tun';
    checks.add(
      _Check(
        tun ? _Level.ok : _Level.warn,
        Icons.shield_rounded,
        tun
            ? _t('همه‌ی برنامه‌ها از وی‌پی‌ان می‌گذرند', 'Every app goes through the VPN')
            : _t('فقط برنامه‌هایی که از پروکسی سیستم پیروی می‌کنند', 'Only apps that follow the system proxy'),
        tun
            ? (leaks['tun_blocks_ipv6'] == true
                  ? _t(
                      'IPv6 بیرون از تونل مسدود است${leaks['tun_blocks_dns'] == true ? ' و DNS بیرون از تونل هم مسدود است' : ''}.',
                      'IPv6 outside the tunnel is blocked${leaks['tun_blocks_dns'] == true ? ', and so is DNS' : ''}.',
                    )
                  : _t('حالت TUN', 'TUN mode'))
            : _t(
                'برنامه‌هایی که پروکسی را نادیده می‌گیرند، و تماس‌های WebRTC/UDP، ممکن است نشانی واقعی را نشان دهند.',
                'Apps that ignore the proxy, and WebRTC/UDP calls, may show the real address.',
              ),
        tun
            ? null
            : (leaks['tun_supported'] == true
                  ? _t('در حالت اتصال، «TUN» را انتخاب کنید.', 'Choose the TUN connection mode.')
                  : _t('در مرورگر، WebRTC را محدود کنید (افزونه یا تنظیمات).', 'Limit WebRTC in the browser.')),
      ),
    );

    checks.add(
      _Check(
        leaks['encrypted_dns'] == true ? _Level.ok : _Level.info,
        Icons.dns_rounded,
        leaks['encrypted_dns'] == true
            ? _t('DNS رمزنگاری‌شده', 'Encrypted DNS')
            : _t('DNS سیستم', 'System DNS'),
        leaks['encrypted_dns'] == true
            ? _t('نام سایت‌ها با DoH/DoT پرسیده می‌شود.', 'Names are looked up over DoH/DoT.')
            : _t('برنامه نام‌ها را با DNS سیستم می‌پرسد.', 'The app looks names up with the system DNS.'),
      ),
    );

    final trackers = r['block_trackers'] == true;
    checks.add(
      _Check(
        trackers ? _Level.ok : _Level.warn,
        Icons.block_rounded,
        trackers
            ? _t('ردیاب‌ها مسدودند', 'Trackers are blocked')
            : _t('ردیاب‌ها مسدود نیستند', 'Trackers are not blocked'),
        _t(
          'ردیاب‌های تبلیغاتی، مستقل از نشانی IP، شما را از سایتی به سایت دیگر دنبال می‌کنند.',
          'Ad trackers follow you from site to site whatever your IP address.',
        ),
        trackers
            ? null
            : _t('«مسدودسازی ردیاب‌ها» را در بالا روشن کنید.', 'Turn on tracker blocking above.'),
      ),
    );

    if (leaks['iran_direct'] == true) {
      checks.add(
        _Check(
          _Level.info,
          Icons.alt_route_rounded,
          _t('سایت‌های ایرانی مستقیم باز می‌شوند', 'Iranian sites go directly'),
          _t(
            'این سایت‌ها نشانی واقعی شما را می‌بینند (عمداً، تا سریع و بی‌مشکل باز شوند)؛ سایت‌های خارجی نه.',
            'They see your real address (on purpose, so they open fast); foreign sites do not.',
          ),
        ),
      );
    }
    return checks;
  }

  String _zoneFix(Map<String, Object?> sync, String exitZone) {
    if (sync['supported'] == true) {
      if (sync['enabled'] != true) {
        return _t(
          '«همگام‌سازی منطقه‌ی زمانی» را در پایین روشن کنید.',
          'Turn on the time zone sync below.',
        );
      }
      if (sync['overridden'] == true) {
        return _t(
          'منطقه‌ی زمانی را دستی عوض کرده‌اید؛ برنامه تا تغییر کشور خروجی به آن دست نمی‌زند.',
          'You changed the zone yourself; it is left alone until the exit changes.',
        );
      }
      return _t(
        'چند ثانیه صبر کنید؛ اگر عوض نشد خطای پایین را ببینید. مرورگر را یک بار ببندید و باز کنید.',
        'Wait a few seconds; restart the browser once.',
      );
    }
    if (Platform.isAndroid) {
      return _t(
        'تنظیمات › سیستم › تاریخ و زمان: «منطقه‌ی زمانی خودکار» را خاموش کنید و $exitZone را انتخاب کنید.',
        'Settings › System › Date & time: turn off automatic time zone and choose $exitZone.',
      );
    }
    if (Platform.isMacOS) {
      return _t(
        'تنظیمات سیستم › عمومی › تاریخ و زمان: منطقه‌ی خودکار را خاموش کنید و $exitZone را انتخاب کنید.',
        'System Settings › General › Date & Time: turn off automatic and choose $exitZone.',
      );
    }
    if (Platform.isIOS) {
      return _t(
        'Settings › General › Date & Time: خودکار را خاموش کنید و $exitZone را انتخاب کنید.',
        'Settings › General › Date & Time: turn off automatic and choose $exitZone.',
      );
    }
    return _t('منطقه‌ی زمانی را دستی روی $exitZone بگذارید.', 'Set the time zone to $exitZone.');
  }

  Color _color(_Level level, ColorScheme c) => switch (level) {
    _Level.ok => const Color(0xFF2E9E5B),
    _Level.warn => const Color(0xFFE09B1A),
    _Level.bad => c.error,
    _Level.info => c.primary,
  };

  IconData _levelIcon(_Level level) => switch (level) {
    _Level.ok => Icons.check_circle_rounded,
    _Level.warn => Icons.warning_amber_rounded,
    _Level.bad => Icons.error_rounded,
    _Level.info => Icons.info_outline_rounded,
  };

  int _score(List<_Check> checks) {
    var total = 0, got = 0;
    for (final c in checks) {
      if (c.level == _Level.info) continue;
      total += 2;
      got += switch (c.level) {
        _Level.ok => 2,
        _Level.warn => 1,
        _ => 0,
      };
    }
    return total == 0 ? 100 : (got * 100 / total).round();
  }

  @override
  Widget build(BuildContext context) {
    final r = _report;
    return Scaffold(
      appBar: AppBar(
        title: Text(_t('سپر هویت', 'Identity Shield')),
        actions: [
          IconButton(
            tooltip: _t('بررسی دوباره', 'Check again'),
            icon: const Icon(Icons.refresh_rounded),
            onPressed: _load,
          ),
        ],
      ),
      body: r == null
          ? Center(
              child: _error == null
                  ? const CircularProgressIndicator()
                  : Padding(
                      padding: const EdgeInsets.all(24),
                      child: Text(_error!, textAlign: TextAlign.center),
                    ),
            )
          : _body(context, r),
    );
  }

  Widget _body(BuildContext context, Map<String, Object?> r) {
    final theme = Theme.of(context);
    final c = theme.colorScheme;
    final checks = _checks(r);
    final score = _score(checks);
    final scoreColor = score >= 80
        ? _color(_Level.ok, c)
        : score >= 50
        ? _color(_Level.warn, c)
        : c.error;
    final sync = _map(r['sync']);
    final browsers = [
      for (final b in (r['browsers'] as List<Object?>? ?? const []))
        if (b is Map<String, Object?>) b,
    ];
    final isDesktop =
        Platform.isWindows || Platform.isLinux || Platform.isMacOS;
    return ListView(
      padding: const EdgeInsets.fromLTRB(16, 8, 16, 32),
      children: [
        Card(
          child: Padding(
            padding: const EdgeInsets.all(16),
            child: Row(
              children: [
                SizedBox(
                  width: 72,
                  height: 72,
                  child: Stack(
                    alignment: Alignment.center,
                    children: [
                      SizedBox.expand(
                        child: CircularProgressIndicator(
                          value: score / 100,
                          strokeWidth: 7,
                          color: scoreColor,
                          backgroundColor: c.surfaceContainerHighest,
                        ),
                      ),
                      Text(
                        '$score',
                        style: theme.textTheme.titleLarge?.copyWith(
                          fontWeight: FontWeight.bold,
                          color: scoreColor,
                        ),
                      ),
                    ],
                  ),
                ),
                const SizedBox(width: 16),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        _t('امتیاز حریم خصوصی این دستگاه', 'This device\'s privacy score'),
                        style: theme.textTheme.titleMedium,
                      ),
                      const SizedBox(height: 4),
                      Text(
                        _t(
                          'نشانی را وی‌پی‌ان پنهان می‌کند؛ این‌ها چیزهای دیگری‌اند که سایت‌ها می‌بینند.',
                          'The VPN hides the address; these are the other things sites see.',
                        ),
                        style: theme.textTheme.bodySmall,
                      ),
                    ],
                  ),
                ),
              ],
            ),
          ),
        ),
        const SizedBox(height: 8),
        if (sync['supported'] == true)
          Card(
            child: Column(
              children: [
                SwitchListTile(
                  secondary: const Icon(Icons.public_rounded),
                  title: Text(_t('همگام‌سازی منطقه‌ی زمانی با کشور خروجی', 'Sync the time zone with the exit country')),
                  subtitle: Text(
                    _t(
                      'هنگام اتصال منطقه‌ی زمانی سیستم را روی کشور سرور می‌گذارد و پس از قطع، منطقه‌ی خودتان را برمی‌گرداند.',
                      'While connected the system time zone follows the server\'s country; yours comes back when disconnected.',
                    ),
                  ),
                  value: sync['enabled'] == true,
                  onChanged: _busy ? null : _setSync,
                ),
                if (_str(sync['applied_zone']).isNotEmpty ||
                    _str(sync['error']).isNotEmpty)
                  Padding(
                    padding: const EdgeInsetsDirectional.fromSTEB(16, 0, 16, 12),
                    child: Align(
                      alignment: AlignmentDirectional.centerStart,
                      child: Text(
                        _str(sync['error']).isNotEmpty
                            ? '${_t('خطا', 'Error')}: ${_str(sync['error'])}'
                            : _t(
                                'اکنون: ${_str(sync['applied_zone'])} — برمی‌گردد به: ${_str(sync['original_zone'])}',
                                'Now: ${_str(sync['applied_zone'])} — restores: ${_str(sync['original_zone'])}',
                              ),
                        style: theme.textTheme.bodySmall?.copyWith(
                          color: _str(sync['error']).isNotEmpty ? c.error : null,
                        ),
                      ),
                    ),
                  ),
              ],
            ),
          ),
        Card(
          child: SwitchListTile(
            secondary: const Icon(Icons.block_rounded),
            title: Text(
              _t(
                'مسدودسازی ردیاب‌ها، بدافزار و فیشینگ',
                'Block trackers, malware and phishing',
              ),
            ),
            subtitle: Text(
              _t(
                'تبلیغات و ردیاب‌های فارسی و خارجی، سایت‌های بدافزار، فیشینگ و استخراج رمزارز برای همه‌ی برنامه‌ها بسته می‌شوند.',
                'Persian and foreign ads and trackers, malware, phishing and mining sites are blocked for every app.',
              ),
            ),
            value: r['block_trackers'] == true,
            onChanged: _busy ? null : _setTrackers,
          ),
        ),
        if (browsers.isNotEmpty || isDesktop) _browserCard(context, browsers),
        for (final check in checks) _checkTile(context, check),
        const SizedBox(height: 12),
        FilledButton.icon(
          icon: const Icon(Icons.travel_explore_rounded),
          label: Text(_t('بررسی آنلاین مرورگر (WebRTC و بیشتر)', 'Online browser check (WebRTC and more)')),
          onPressed: () => openUrl(identityCheckUrl),
        ),
        const SizedBox(height: 8),
        Text(
          _t(
            'این بررسی را در همان مرورگری باز کنید که با آن وب‌گردی می‌کنید؛ نشت WebRTC و اثر انگشت مرورگر را فقط خود مرورگر نشان می‌دهد.',
            'Open it in the browser you browse with: only the browser itself shows WebRTC leaks and its fingerprint.',
          ),
          style: theme.textTheme.bodySmall,
          textAlign: TextAlign.center,
        ),
      ],
    );
  }

  Widget _browserCard(
    BuildContext context,
    List<Map<String, Object?>> browsers,
  ) {
    final theme = Theme.of(context);
    final connected = _report?['connected'] == true;
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                const Icon(Icons.privacy_tip_rounded),
                const SizedBox(width: 12),
                Expanded(
                  child: Text(
                    _t('مرورگر خصوصی', 'Private browser'),
                    style: theme.textTheme.titleMedium,
                  ),
                ),
              ],
            ),
            const SizedBox(height: 6),
            Text(
              _t(
                'یک پروفایل جدا که همه‌چیزش فقط از وی‌پی‌ان می‌گذرد (DNS هم)، WebRTC نشت نمی‌دهد، زبانش انگلیسی است و اگر وی‌پی‌ان قطع شود اصلاً وصل نمی‌شود. در فایرفاکس، ضداثرانگشت هم روشن است (مثل Tor Browser: منطقه‌ی زمانی UTC، فونت‌ها و Canvas یکسان).',
                'A separate profile: everything goes only through the VPN (DNS too), no WebRTC leaks, English language, and no connection at all if the VPN drops. In Firefox fingerprinting resistance is on as well (like Tor Browser).',
              ),
              style: theme.textTheme.bodySmall,
            ),
            const SizedBox(height: 12),
            if (browsers.isEmpty)
              Text(
                _t(
                  'مرورگری پیدا نشد. فایرفاکس را نصب کنید (بهترین گزینه).',
                  'No browser found. Install Firefox (the best choice).',
                ),
                style: theme.textTheme.bodySmall,
              )
            else
              Wrap(
                spacing: 8,
                runSpacing: 8,
                children: [
                  for (final b in browsers)
                    FilledButton.tonalIcon(
                      icon: const Icon(Icons.open_in_new_rounded, size: 18),
                      label: Text(_str(b['name'])),
                      onPressed: connected
                          ? () => _launch(_str(b['id']))
                          : null,
                    ),
                ],
              ),
            if (!connected && browsers.isNotEmpty) ...[
              const SizedBox(height: 8),
              Text(
                _t('اول وصل شوید.', 'Connect first.'),
                style: theme.textTheme.bodySmall,
              ),
            ],
          ],
        ),
      ),
    );
  }

  Widget _checkTile(BuildContext context, _Check check) {
    final theme = Theme.of(context);
    final color = _color(check.level, theme.colorScheme);
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(14),
        child: Row(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Icon(check.icon, color: color),
            const SizedBox(width: 12),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Row(
                    children: [
                      Expanded(
                        child: Text(check.title, style: theme.textTheme.titleSmall),
                      ),
                      Icon(_levelIcon(check.level), size: 18, color: color),
                    ],
                  ),
                  const SizedBox(height: 4),
                  Text(check.detail, style: theme.textTheme.bodySmall),
                  if (check.fix != null) ...[
                    const SizedBox(height: 6),
                    Text(
                      check.fix!,
                      style: theme.textTheme.bodySmall?.copyWith(color: color),
                    ),
                  ],
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}
