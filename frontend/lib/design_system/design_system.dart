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
        mutedForeground: Color(0xFF52525B),
      ),
    ),
    darkTheme: ShadThemeData(
      brightness: Brightness.dark,
      radius: BorderRadius.circular(BPTokens.radius),
      colorScheme: const ShadZincColorScheme.dark(
        background: BPTokens.canvasDark,
        card: BPTokens.surfaceDark,
        primary: BPTokens.primaryDark,
        primaryForeground: BPTokens.onPrimaryDark,
        ring: BPTokens.primaryDark,
      ),
    ),
    home: home,
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
      theme.textTheme.h3,
      theme.textTheme.large,
      theme.textTheme.muted,
      theme.textTheme.small,
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
                  ExcludeSemantics(child: Icon(_navIcon(navIcon!), size: 20)),
                  const SizedBox(height: BPTokens.xs),
                  Text(label, textAlign: TextAlign.center),
                ],
              ),
      );
      return ConstrainedBox(
        constraints: const BoxConstraints(
          minHeight: BPTokens.touchTarget,
          minWidth: BPTokens.touchTarget,
        ),
        child: ShadButton.raw(
          variant: primary
              ? ShadButtonVariant.primary
              : ShadButtonVariant.outline,
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
