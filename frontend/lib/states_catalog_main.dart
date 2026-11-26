import "package:flutter/widgets.dart";

import "design_system/design_system.dart";
import "design_system/bp_view_states.dart";

void main() => runApp(const BPApp(home: BPStatesExample()));

/// This catalog renders fixtures only; no network or persistence.
class BPStatesExample extends StatefulWidget {
  const BPStatesExample({super.key});
  @override
  State<BPStatesExample> createState() => _BPStatesExampleState();
}

class _BPStatesExampleState extends State<BPStatesExample> {
  int actions = 0;
  void _exampleAction() => setState(() => actions++);

  @override
  Widget build(BuildContext context) {
    final theme = BPVisuals.of(context);
    return ColoredBox(
      color: theme.background,
      child: SafeArea(
        child: ListView(
          padding: const EdgeInsets.all(BPTokens.lg),
          children: [
            Text("Catálogo de estados · Simulación", style: theme.heading),
            const SizedBox(height: BPTokens.md),
            Text("Acciones de ejemplo: $actions", style: theme.label),
            const SizedBox(height: BPTokens.lg),
            const BPLoadingState(),
            const SizedBox(height: BPTokens.lg),
            BPEmptyState(onAction: _exampleAction),
            const SizedBox(height: BPTokens.lg),
            BPErrorState(onRetry: _exampleAction),
            const SizedBox(height: BPTokens.lg),
            BPOfflineState(onRetry: _exampleAction),
          ],
        ),
      ),
    );
  }
}
