import "package:flutter/widgets.dart";

import "design_system.dart";

enum BPTone { neutral, success, warning, danger }

/// Semantic labels with explicit text; color is never the only signal.
class BPBadge extends StatelessWidget {
  const BPBadge({super.key, required this.label, required this.tone});
  final String label;
  final BPTone tone;

  static (Color, Color) colors(BPTone tone, Brightness brightness) {
    final dark = brightness == Brightness.dark;
    return switch (tone) {
      BPTone.neutral =>
        dark
            ? (const Color(0xFF334155), const Color(0xFFF1F5F9))
            : (const Color(0xFFE2E8F0), const Color(0xFF334155)),
      BPTone.success =>
        dark
            ? (const Color(0xFF14532D), const Color(0xFFDCFCE7))
            : (const Color(0xFFDCFCE7), const Color(0xFF166534)),
      BPTone.warning =>
        dark
            ? (const Color(0xFF78350F), const Color(0xFFFEF3C7))
            : (const Color(0xFFFEF3C7), const Color(0xFF92400E)),
      BPTone.danger =>
        dark
            ? (const Color(0xFF7F1D1D), const Color(0xFFFEE2E2))
            : (const Color(0xFFFEE2E2), const Color(0xFF991B1B)),
    };
  }

  @override
  Widget build(BuildContext context) {
    final theme = BPVisuals.of(context);
    final (background, foreground) = colors(tone, theme.brightness);
    return DecoratedBox(
      decoration: BoxDecoration(
        color: background,
        borderRadius: BorderRadius.circular(BPTokens.radius),
      ),
      child: Padding(
        padding: const EdgeInsets.symmetric(
          horizontal: BPTokens.sm,
          vertical: BPTokens.xs,
        ),
        child: Text(
          label,
          style: theme.label.copyWith(
            color: foreground,
            fontWeight: FontWeight.w600,
          ),
        ),
      ),
    );
  }
}
