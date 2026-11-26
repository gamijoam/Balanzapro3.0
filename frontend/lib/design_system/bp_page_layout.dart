import "package:flutter/widgets.dart";

import "design_system.dart";

/// Navigation entries are presentation only, never permission grants.
class BPNavItem {
  const BPNavItem(this.id, this.label, {this.icon});
  final BPNavIcon? icon;
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
    final keyboardInset = MediaQuery.viewInsetsOf(context).bottom;
    return ColoredBox(
      color: theme.background,
      child: Padding(
        padding: EdgeInsets.only(bottom: keyboardInset),
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
                padding: const EdgeInsets.symmetric(
                  horizontal: BPTokens.lg,
                  vertical: BPTokens.sm,
                ),
                decoration: BoxDecoration(
                  color: theme.surface,
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
                  if (!wide && keyboardInset == 0)
                    ConstrainedBox(
                      constraints: BoxConstraints(
                        maxHeight: constraints.maxHeight / 3,
                      ),
                      child: navigation,
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
            navIcon: item.icon,
            onPressed: () => onNavigate(item.id),
          ),
        ),
    ];
    return FocusTraversalGroup(
      child: ColoredBox(
        color: BPVisuals.of(context).surface,
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
                    for (final button in buttons)
                      Padding(
                        padding: const EdgeInsets.only(bottom: BPTokens.sm),
                        child: button,
                      ),
                  ],
                ),
        ),
      ),
    );
  }
}
