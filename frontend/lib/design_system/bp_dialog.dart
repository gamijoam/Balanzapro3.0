import "package:flutter/services.dart";
import "package:flutter/widgets.dart";

import "design_system.dart";

/// Presentation-only confirmation. Never executes the operation.
/// Only the explicit confirmation button returns true; every dismissal
/// (including system back) returns false.
abstract final class BPDialog {
  static Future<bool> confirm(
    BuildContext context, {
    required String title,
    required String description,
    required String confirmLabel,
  }) async {
    if (title.trim().isEmpty ||
        description.trim().isEmpty ||
        confirmLabel.trim().isEmpty ||
        confirmLabel == "Cancelar") {
      throw ArgumentError("Invalid confirmation labels");
    }
    return await showGeneralDialog<bool>(
          context: context,
          barrierDismissible: true,
          barrierLabel: "Cancelar confirmación",
          transitionDuration: Duration.zero,
          pageBuilder: (context, _, _) => _Confirmation(
            title: title,
            description: description,
            confirmLabel: confirmLabel,
          ),
        ) ??
        false;
  }
}

class _Confirmation extends StatefulWidget {
  const _Confirmation({
    required this.title,
    required this.description,
    required this.confirmLabel,
  });
  final String title, description, confirmLabel;
  @override
  State<_Confirmation> createState() => _ConfirmationState();
}

class _ConfirmationState extends State<_Confirmation> {
  bool closing = false;
  void _finish(bool confirmed) {
    if (closing || ModalRoute.of(context)?.isCurrent != true) return;
    closing = true;
    Navigator.of(context).pop(confirmed);
  }

  @override
  Widget build(BuildContext context) {
    final theme = BPVisuals.of(context);
    return CallbackShortcuts(
      bindings: {
        const SingleActivator(LogicalKeyboardKey.escape): () => _finish(false),
      },
      child: FocusTraversalGroup(
        child: SafeArea(
          child: Padding(
            padding: const EdgeInsets.all(BPTokens.lg),
            child: Center(
              child: ConstrainedBox(
                constraints: const BoxConstraints(
                  maxWidth: BPTokens.dialogMaxWidth,
                ),
                child: SingleChildScrollView(
                  child: Semantics(
                    scopesRoute: true,
                    explicitChildNodes: true,
                    namesRoute: true,
                    label: widget.title,
                    child: BPCard(
                      child: Column(
                        mainAxisSize: MainAxisSize.min,
                        crossAxisAlignment: CrossAxisAlignment.stretch,
                        children: [
                          Text(widget.title, style: theme.heading),
                          const SizedBox(height: BPTokens.sm),
                          Text(widget.description, style: theme.label),
                          const SizedBox(height: BPTokens.xl),
                          BPButton(
                            label: "Cancelar",
                            primary: false,
                            autofocus: true,
                            onPressed: () => _finish(false),
                          ),
                          const SizedBox(height: BPTokens.sm),
                          BPButton(
                            label: widget.confirmLabel,
                            onPressed: () => _finish(true),
                          ),
                        ],
                      ),
                    ),
                  ),
                ),
              ),
            ),
          ),
        ),
      ),
    );
  }
}
