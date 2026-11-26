import "package:flutter/services.dart";
import "package:flutter/widgets.dart";
import "package:shadcn_ui/shadcn_ui.dart";

/// Central provisional design values, independent of feature modules.
abstract final class BPTokens {
  static const double xs = 4, sm = 8, md = 12, lg = 16, xl = 24, xxl = 32;
  static const double touchTarget = 48,
      wideBreakpoint = 1000,
      sidebarWidth = 224;
  static const double radius = 10;
  static const double headingSize = 22,
      titleSize = 16,
      labelSize = 14,
      mutedSize = 13;
  static const double navIconSize = 20;
  static const Color inkLight = Color(0xFF17243A);
  static const Color mutedLight = Color(0xFF526177);
  static const Color inkDark = Color(0xFFF1F5F9);
  static const Color mutedDark = Color(0xFFB6C4D6);
  static const Color navActiveLight = Color(0xFFDBEAFE);
  static const Color navActiveDark = Color(0xFF253D63);

  static (Color, Color) navigationColors(Brightness brightness) =>
      brightness == Brightness.dark
      ? (navActiveDark, primaryDark)
      : (navActiveLight, primaryLight);

  static const Color canvasLight = Color(0xFFF5F7FB);
  static const Color canvasDark = Color(0xFF111827);
  static const Color surfaceDark = Color(0xFF1E293B);
  static const double formMaxWidth = 720;
  static const double dialogMaxWidth = 480;
  static const Color primaryLight = Color(0xFF1E40AF);
  static const Color onPrimaryLight = Color(0xFFFFFFFF);
  static const Color primaryDark = Color(0xFF93C5FD);
  static const Color onPrimaryDark = Color(0xFF172554);
}

class BPApp extends StatelessWidget {
  const BPApp({super.key, required this.home});
  final Widget home;

  @override
  Widget build(BuildContext context) => ShadApp(
    title: "BalanzaPro3.0",
    theme: ShadThemeData(
      brightness: Brightness.light,
      radius: BorderRadius.circular(BPTokens.radius),
      colorScheme: const ShadZincColorScheme.light(
        background: BPTokens.canvasLight,
        primary: BPTokens.primaryLight,
        primaryForeground: BPTokens.onPrimaryLight,
        ring: BPTokens.primaryLight,
        foreground: BPTokens.inkLight,
        cardForeground: BPTokens.inkLight,
        mutedForeground: BPTokens.mutedLight,
      ),
    ),
    darkTheme: ShadThemeData(
      brightness: Brightness.dark,
      radius: BorderRadius.circular(BPTokens.radius),
      colorScheme: const ShadZincColorScheme.dark(
        background: BPTokens.canvasDark,
        card: BPTokens.surfaceDark,
        foreground: BPTokens.inkDark,
        cardForeground: BPTokens.inkDark,
        mutedForeground: BPTokens.mutedDark,
        primary: BPTokens.primaryDark,
        primaryForeground: BPTokens.onPrimaryDark,
        ring: BPTokens.primaryDark,
      ),
    ),
    home: Builder(
      builder: (context) {
        final theme = ShadTheme.of(context);
        return DefaultTextStyle(
          style: theme.textTheme.p.copyWith(
            fontSize: BPTokens.labelSize,
            height: 1.4,
            color: theme.colorScheme.foreground,
          ),
          child: home,
        );
      },
    ),
  );
}

/// Provider-neutral view of semantic theme values.
class BPVisuals {
  const BPVisuals._(
    this.brightness,
    this.background,
    this.surface,
    this.border,
    this.heading,
    this.title,
    this.muted,
    this.label,
  );
  final Brightness brightness;
  final Color background, surface, border;
  final TextStyle heading, title, muted, label;
  static BPVisuals of(BuildContext context) {
    final theme = ShadTheme.of(context);
    return BPVisuals._(
      theme.brightness,
      theme.colorScheme.background,
      theme.colorScheme.card,
      theme.colorScheme.border,
      theme.textTheme.h3.copyWith(
        fontSize: BPTokens.headingSize,
        height: 1.25,
        fontWeight: FontWeight.w700,
      ),
      theme.textTheme.large.copyWith(
        fontSize: BPTokens.titleSize,
        height: 1.35,
        fontWeight: FontWeight.w600,
      ),
      theme.textTheme.muted.copyWith(
        fontSize: BPTokens.mutedSize,
        height: 1.45,
        color: theme.colorScheme.mutedForeground,
      ),
      theme.textTheme.small.copyWith(fontSize: BPTokens.labelSize, height: 1.4),
    );
  }
}

enum BPNavIcon { home, products, movements }

IconData _navIcon(BPNavIcon icon) => switch (icon) {
  BPNavIcon.home => LucideIcons.house,
  BPNavIcon.products => LucideIcons.package,
  BPNavIcon.movements => LucideIcons.arrowLeftRight,
};

class BPButton extends StatelessWidget {
  const BPButton({
    super.key,
    required this.label,
    required this.onPressed,
    this.primary = true,
    this.autofocus = false,
    this.navIcon,
  });
  final String label;
  final VoidCallback? onPressed;
  final bool primary, autofocus;
  final BPNavIcon? navIcon;
  @override
  Widget build(BuildContext context) => LayoutBuilder(
    builder: (context, constraints) {
      final isNav = navIcon != null;
      final visual = BPVisuals.of(context);
      final (navBackground, navForeground) = BPTokens.navigationColors(
        visual.brightness,
      );
      const padding = EdgeInsets.symmetric(
        horizontal: BPTokens.md,
        vertical: BPTokens.sm,
      );
      // ShadButton lays out its label in a non-flex Row. Bound the label
      // explicitly and permit vertical growth for enlarged text.
      final child = ConstrainedBox(
        constraints: BoxConstraints(
          maxWidth: constraints.hasBoundedWidth
              ? (constraints.maxWidth - padding.horizontal - BPTokens.xs).clamp(
                  0.0,
                  double.infinity,
                )
              : double.infinity,
        ),
        child: navIcon == null
            ? Text(label, textAlign: TextAlign.center)
            : Column(
                mainAxisSize: MainAxisSize.min,
                children: [
                  ExcludeSemantics(
                    child: DecoratedBox(
                      key: primary ? const ValueKey("bp-nav-active") : null,
                      decoration: BoxDecoration(
                        color: primary ? navBackground : null,
                        borderRadius: BorderRadius.circular(BPTokens.radius),
                      ),
                      child: Padding(
                        padding: const EdgeInsets.symmetric(
                          horizontal: BPTokens.md,
                          vertical: BPTokens.xs,
                        ),
                        child: Icon(
                          _navIcon(navIcon!),
                          size: BPTokens.navIconSize,
                        ),
                      ),
                    ),
                  ),
                  const SizedBox(height: BPTokens.xs),
                  Text(
                    label,
                    textAlign: TextAlign.center,
                    style: TextStyle(
                      fontSize: BPTokens.mutedSize,
                      fontWeight: primary ? FontWeight.w700 : FontWeight.w500,
                    ),
                  ),
                ],
              ),
      );
      return ConstrainedBox(
        constraints: const BoxConstraints(
          minHeight: BPTokens.touchTarget,
          minWidth: BPTokens.touchTarget,
        ),
        child: ShadButton.raw(
          variant: isNav
              ? ShadButtonVariant.ghost
              : primary
              ? ShadButtonVariant.primary
              : ShadButtonVariant.outline,
          foregroundColor: isNav
              ? (primary
                    ? navForeground
                    : (visual.brightness == Brightness.dark
                          ? BPTokens.mutedDark
                          : BPTokens.mutedLight))
              : null,
          hoverForegroundColor: isNav ? navForeground : null,
          hoverBackgroundColor: isNav ? navBackground : null,

          height: 0,
          autofocus: autofocus,
          padding: padding,
          enabled: onPressed != null,
          onPressed: onPressed,
          child: child,
        ),
      );
    },
  );
}

class BPInput extends StatelessWidget {
  const BPInput({
    super.key,
    required this.controller,
    required this.label,
    required this.placeholder,
    required this.onChanged,
    this.enabled = true,
    this.focusNode,
    this.textInputAction,
    this.onSubmitted,
  });
  final TextEditingController controller;
  final String label, placeholder;
  final ValueChanged<String> onChanged;
  final bool enabled;
  final FocusNode? focusNode;
  final TextInputAction? textInputAction;
  final ValueChanged<String>? onSubmitted;
  @override
  Widget build(BuildContext context) => Semantics(
    label: label,
    child: ConstrainedBox(
      constraints: const BoxConstraints(minHeight: BPTokens.touchTarget),
      child: ShadInput(
        controller: controller,
        enabled: enabled,
        focusNode: focusNode,
        textInputAction: textInputAction,
        onSubmitted: onSubmitted,
        placeholder: Text(placeholder),
        onChanged: onChanged,
      ),
    ),
  );
}

class BPCard extends StatelessWidget {
  const BPCard({super.key, required this.child, this.compact = false});
  final Widget child;
  final bool compact;
  @override
  Widget build(BuildContext context) => ShadCard(
    padding: EdgeInsets.all(compact ? BPTokens.md : BPTokens.lg),
    shadows: const [],
    child: child,
  );
}

/// Non-animated placeholder. The enclosing loading region announces status.
class BPSkeleton extends StatelessWidget {
  const BPSkeleton({super.key, required this.width, required this.height});
  final double width, height;
  @override
  Widget build(BuildContext context) => ExcludeSemantics(
    child: Container(
      width: width,
      height: height,
      decoration: BoxDecoration(
        color: ShadTheme.of(context).colorScheme.muted,
        borderRadius: BorderRadius.circular(BPTokens.radius),
      ),
    ),
  );
}
