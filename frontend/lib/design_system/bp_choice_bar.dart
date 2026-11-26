import "package:flutter/widgets.dart";

import "design_system.dart";

class BPChoice {
  const BPChoice(this.id, this.label);
  final String id, label;
}

/// Caller-controlled single choice, with keyboard-accessible shared buttons.
/// The choices wrap at narrow widths and enlarged text.
class BPChoiceBar extends StatelessWidget {
  BPChoiceBar({
    super.key,
    required List<BPChoice> choices,
    required this.selectedId,
    required this.onChanged,
    required this.label,
    this.keyPrefix = "bp-choice",
  }) : choices = List.unmodifiable(choices) {
    if (choices.isEmpty ||
        choices.length > 8 ||
        choices.any((c) => c.id.trim().isEmpty || c.label.trim().isEmpty) ||
        choices.map((c) => c.id).toSet().length != choices.length ||
        !choices.any((c) => c.id == selectedId)) {
      throw ArgumentError("Invalid choice group");
    }
  }

  final List<BPChoice> choices;
  final String selectedId, label, keyPrefix;
  final ValueChanged<String> onChanged;

  @override
  Widget build(BuildContext context) => FocusTraversalGroup(
    child: Semantics(
      container: true,
      label: label,
      child: Wrap(
        spacing: BPTokens.sm,
        runSpacing: BPTokens.sm,
        children: [
          for (final choice in choices)
            Semantics(
              selected: choice.id == selectedId,
              inMutuallyExclusiveGroup: true,
              child: BPButton(
                key: ValueKey("$keyPrefix-${choice.id}"),
                label: choice.label,
                primary: choice.id == selectedId,
                onPressed: () => onChanged(choice.id),
              ),
            ),
        ],
      ),
    ),
  );
}
