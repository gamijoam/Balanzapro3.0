import "package:flutter/widgets.dart";

import "design_system.dart";
import "bp_adaptive.dart";

import "package:shadcn_ui/shadcn_ui.dart";

/// Navigation entries are presentation only, never permission grants.
class BPNavItem {
  const BPNavItem(this.id, this.label, {this.icon});
  final BPNavIcon? icon;
  final String id, label;
}

/// Caller-controlled layout. Content must manage its own scrolling.
class BPPageLayout extends StatefulWidget {
  BPPageLayout({
    super.key,
    required this.title,
    required this.contextLabel,
    required List<BPNavItem> items,
    required this.selectedId,
    required this.onNavigate,
    required this.child,
  }) : items = List.unmodifiable(items) {
    if (items.isEmpty ||
        items.length > 12 ||
        items.any(
          (item) => item.id.trim().isEmpty || item.label.trim().isEmpty,
        ) ||
        items.map((item) => item.id).toSet().length != items.length ||
        !items.any((item) => item.id == selectedId)) {
      throw ArgumentError("Invalid navigation");
    }
  }

  final String title, contextLabel, selectedId;
  final List<BPNavItem> items;
  final ValueChanged<String> onNavigate;
  final Widget child;

  @override
  State<BPPageLayout> createState() => _BPPageLayoutState();
}

class _BPPageLayoutState extends State<BPPageLayout> {
  bool collapsed = false;

  @override
  Widget build(BuildContext context) {
    final theme = BPVisuals.of(context);
    final keyboardInset = MediaQuery.viewInsetsOf(context).bottom;
    return ColoredBox(
      color: theme.background,
      child: Padding(
        padding: EdgeInsets.only(bottom: keyboardInset),
        child: SafeArea(
          child: LayoutBuilder(
            builder: (context, constraints) {
              final mode = BPAdaptive.mode(
                constraints.maxWidth,
                MediaQuery.textScalerOf(context),
              );
              final compact = mode == BPLayoutMode.compact;
              final rail = mode == BPLayoutMode.rail || collapsed;
              final nav = BPSidebar(
                key: Key(switch (mode) {
                  BPLayoutMode.compact => "compact-navigation",
                  BPLayoutMode.rail => "rail-navigation",
                  BPLayoutMode.expanded => "desktop-navigation",
                }),
                items: widget.items,
                selectedId: widget.selectedId,
                onNavigate: widget.onNavigate,
                compact: compact,
                rail: rail,
                onToggle: mode == BPLayoutMode.expanded
                    ? () => setState(() => collapsed = !collapsed)
                    : null,
              );
              final header = Container(
                width: double.infinity,
                padding: const EdgeInsets.symmetric(
                  horizontal: BPTokens.lg,
                  vertical: BPTokens.sm,
                ),
                decoration: BoxDecoration(
                  color: theme.surface,
                  border: Border(bottom: BorderSide(color: theme.border)),
                ),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(widget.title, style: theme.title),
                    Text(widget.contextLabel, style: theme.muted),
                  ],
                ),
              );
              return Row(
                key: const Key("bp-page-layout"),
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  if (!compact)
                    SizedBox(
                      width: BPAdaptive.sideWidth(constraints.maxWidth, rail),
                      child: nav,
                    ),
                  Expanded(
                    key: const ValueKey("bp-workspace"),
                    child: Column(
                      children: [
                        ConstrainedBox(
                          constraints: BoxConstraints(
                            maxHeight: constraints.maxHeight / 3,
                          ),
                          child: SingleChildScrollView(child: header),
                        ),
                        Expanded(child: widget.child),
                        if (compact && keyboardInset == 0)
                          ConstrainedBox(
                            constraints: BoxConstraints(
                              maxHeight: constraints.maxHeight / 3,
                            ),
                            child: nav,
                          ),
                      ],
                    ),
                  ),
                ],
              );
            },
          ),
        ),
      ),
    );
  }
}

class BPSidebar extends StatelessWidget {
  const BPSidebar({
    super.key,
    required this.items,
    required this.selectedId,
    required this.onNavigate,
    required this.compact,
    this.rail = false,
    this.onToggle,
  });
  final List<BPNavItem> items;
  final String selectedId;
  final ValueChanged<String> onNavigate;
  final bool compact, rail;
  final VoidCallback? onToggle;

  @override
  Widget build(BuildContext context) {
    final theme = BPVisuals.of(context);
    final buttons = [
      for (final item in items)
        Semantics(
          selected: item.id == selectedId,
          child: compact
              ? BPButton(
                  key: ValueKey("bp-nav-${item.id}"),
                  label: item.label,
                  primary: item.id == selectedId,
                  navIcon: item.icon,
                  onPressed: () => onNavigate(item.id),
                )
              : _SideDestination(
                  key: ValueKey("bp-nav-${item.id}"),
                  item: item,
                  active: item.id == selectedId,
                  rail: rail,
                  onPressed: () => onNavigate(item.id),
                ),
        ),
    ];
    return FocusTraversalGroup(
      child: Container(
        decoration: BoxDecoration(
          color: theme.surface,
          border: Border(
            right: compact ? BorderSide.none : BorderSide(color: theme.border),
            top: compact ? BorderSide(color: theme.border) : BorderSide.none,
          ),
        ),
        child: SingleChildScrollView(
          padding: const EdgeInsets.all(BPTokens.sm),
          child: compact
              ? LayoutBuilder(
                  builder: (context, constraints) {
                    if (buttons.length > 4 ||
                        MediaQuery.textScalerOf(context).scale(14) > 21) {
                      return Wrap(
                        spacing: BPTokens.sm,
                        runSpacing: BPTokens.sm,
                        children: buttons,
                      );
                    }
                    return Row(
                      children: [
                        for (final button in buttons)
                          Expanded(
                            child: Padding(
                              padding: const EdgeInsets.symmetric(
                                horizontal: BPTokens.xs,
                              ),
                              child: button,
                            ),
                          ),
                      ],
                    );
                  },
                )
              : Column(
                  crossAxisAlignment: CrossAxisAlignment.stretch,
                  children: [
                    if (onToggle != null) ...[
                      _SideDestination(
                        key: const ValueKey("bp-nav-toggle"),
                        item: BPNavItem(
                          "toggle",
                          rail ? "Expandir menú" : "Contraer menú",
                        ),
                        active: false,
                        rail: rail,
                        onPressed: onToggle!,
                        toggle: true,
                      ),
                      const SizedBox(height: BPTokens.lg),
                    ],
                    for (final button in buttons)
                      Padding(
                        padding: const EdgeInsets.only(bottom: BPTokens.xs),
                        child: button,
                      ),
                  ],
                ),
        ),
      ),
    );
  }
}

/// One shared focusable destination with an actual touch target. The rail
/// retains visible labels rather than depending on hover/tooltips.
class _SideDestination extends StatelessWidget {
  const _SideDestination({
    super.key,
    required this.item,
    required this.active,
    required this.rail,
    required this.onPressed,
    this.toggle = false,
  });
  final BPNavItem item;
  final bool active, rail, toggle;
  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) {
    final scheme = ShadTheme.of(context).colorScheme;
    final color = active ? scheme.primary : scheme.mutedForeground;
    final icon = toggle
        ? (rail ? LucideIcons.panelLeftOpen : LucideIcons.panelLeftClose)
        : switch (item.icon) {
            BPNavIcon.home => LucideIcons.house,
            BPNavIcon.products => LucideIcons.package,
            BPNavIcon.movements => LucideIcons.arrowLeftRight,
            null => LucideIcons.menu,
          };
    final children = [
      ExcludeSemantics(child: Icon(icon, size: 20)),
      SizedBox(width: rail ? 0 : BPTokens.sm, height: rail ? BPTokens.xs : 0),
      if (rail)
        Text(
          item.label,
          textAlign: TextAlign.center,
          style: const TextStyle(fontSize: 12),
        )
      else
        Expanded(
          child: Text(
            item.label,
            textAlign: TextAlign.start,
            style: const TextStyle(fontSize: 14),
          ),
        ),
    ];
    return ConstrainedBox(
      constraints: const BoxConstraints(minHeight: BPTokens.touchTarget),
      child: ShadButton.raw(
        key: active ? const ValueKey("bp-nav-active") : null,
        variant: ShadButtonVariant.ghost,
        height: 0,
        padding: EdgeInsets.symmetric(
          horizontal: rail ? BPTokens.xs : BPTokens.md,
          vertical: BPTokens.sm,
        ),
        backgroundColor: active ? scheme.primary.withValues(alpha: .12) : null,
        foregroundColor: color,
        hoverForegroundColor: scheme.primary,
        expands: true,
        onPressed: onPressed,
        child: rail
            ? Column(mainAxisSize: MainAxisSize.min, children: children)
            : Row(children: children),
      ),
    );
  }
}
