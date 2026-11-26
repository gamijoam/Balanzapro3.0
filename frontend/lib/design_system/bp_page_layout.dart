import "package:flutter/widgets.dart";

import "design_system.dart";

/// Navigation entries are presentation only, never permission grants.
class BPNavItem {
  const BPNavItem(this.id, this.label);
  final String id, label;
}

/// Caller-controlled layout. Content must manage its own scrolling.
class BPPageLayout extends StatelessWidget {
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
  Widget build(BuildContext context) {
    final theme = BPVisuals.of(context);
    return ColoredBox(
      color: theme.background,
      child: SafeArea(
        child: LayoutBuilder(
          builder: (context, constraints) {
            final wide =
                constraints.maxWidth >= BPTokens.wideBreakpoint &&
                MediaQuery.textScalerOf(context).scale(14) <= 21;
            final navigation = BPSidebar(
              key: Key(wide ? "desktop-navigation" : "compact-navigation"),
              items: items,
              selectedId: selectedId,
              onNavigate: onNavigate,
              compact: !wide,
            );
            final header = Container(
              padding: const EdgeInsets.all(BPTokens.md),
              decoration: BoxDecoration(
                border: Border(bottom: BorderSide(color: theme.border)),
              ),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  Text(title, style: theme.title),
                  Text(contextLabel, style: theme.muted),
                ],
              ),
            );
            return Column(
              key: const Key("bp-page-layout"),
              children: [
                // On short windows the header can scroll instead of crushing
                // the content viewport. It retains its complete context label.
                ConstrainedBox(
                  constraints: BoxConstraints(
                    maxHeight: constraints.maxHeight / 3,
                  ),
                  child: SingleChildScrollView(child: header),
                ),
                if (!wide)
                  ConstrainedBox(
                    constraints: BoxConstraints(
                      maxHeight: constraints.maxHeight / 3,
                    ),
                    child: navigation,
                  ),
                Expanded(
                  child: Row(
                    children: [
                      if (wide)
                        SizedBox(
                          width: BPTokens.sidebarWidth,
                          child: navigation,
                        ),
                      Expanded(child: child),
                    ],
                  ),
                ),
              ],
            );
          },
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
  });
  final List<BPNavItem> items;
  final String selectedId;
  final ValueChanged<String> onNavigate;
  final bool compact;

  @override
  Widget build(BuildContext context) {
    final buttons = [
      for (final item in items)
        Semantics(
          selected: item.id == selectedId,
          child: BPButton(
            key: ValueKey("bp-nav-${item.id}"),
            label: item.label,
            primary: item.id == selectedId,
            onPressed: () => onNavigate(item.id),
          ),
        ),
    ];
    return FocusTraversalGroup(
      child: SingleChildScrollView(
        padding: const EdgeInsets.all(BPTokens.sm),
        child: compact
            ? Wrap(
                spacing: BPTokens.sm,
                runSpacing: BPTokens.sm,
                children: buttons,
              )
            : Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  for (final button in buttons)
                    Padding(
                      padding: const EdgeInsets.only(bottom: BPTokens.sm),
                      child: button,
                    ),
                ],
              ),
      ),
    );
  }
}
