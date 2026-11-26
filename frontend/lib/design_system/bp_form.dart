import "package:flutter/services.dart";
import "package:flutter/widgets.dart";

import "design_system.dart";

/// Presentation validation only. Controllers are owned by the caller.
class BPTextField {
  const BPTextField({
    required this.id,
    required this.label,
    required this.controller,
    this.placeholder = "",
    this.validate,
  });
  final String id, label, placeholder;
  final TextEditingController controller;
  final String? Function(String)? validate;
}

/// One in-flight callback per mounted form. A true response means the
/// caller confirmed success; false means a confirmed rejection, not an
/// uncertain outcome. An exception locks this instance until the caller
/// reconciles and remounts it. Input is retained.
/// Business validation, durable idempotency and unknown outcomes belong
/// to the application layer, not this widget.
class BPForm extends StatefulWidget {
  BPForm({
    super.key,
    required List<BPTextField> fields,
    required this.onSubmit,
    this.successMessage = "Cambios guardados",
  }) : fields = List.unmodifiable(fields) {
    if (fields.isEmpty ||
        fields.length > 12 ||
        fields.any((f) => f.id.trim().isEmpty || f.label.trim().isEmpty) ||
        fields.map((f) => f.id).toSet().length != fields.length ||
        fields.map((f) => f.controller).toSet().length != fields.length) {
      throw ArgumentError("Invalid BPForm fields");
    }
  }
  final List<BPTextField> fields;
  final Future<bool> Function(Map<String, String>) onSubmit;
  final String successMessage;

  @override
  State<BPForm> createState() => _BPFormState();
}

class _BPFormState extends State<BPForm> {
  final focus = <String, FocusNode>{};
  final errors = <String, String>{};
  bool busy = false, attempted = false, uncertain = false;
  String? message;
  int revision = 0;

  @override
  void initState() {
    super.initState();
    _syncFocus();
  }

  void _syncFocus() {
    for (final field in widget.fields) {
      focus.putIfAbsent(field.id, () => FocusNode(debugLabel: field.label));
    }
    final removed = focus.keys
        .where((id) => !widget.fields.any((f) => f.id == id))
        .toList();
    for (final id in removed) {
      focus.remove(id)?.dispose();
      errors.remove(id);
    }
  }

  @override
  void didUpdateWidget(covariant BPForm oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.fields.length != widget.fields.length ||
        List.generate(widget.fields.length, (i) => i).any(
          (i) =>
              i >= oldWidget.fields.length ||
              oldWidget.fields[i].id != widget.fields[i].id ||
              oldWidget.fields[i].controller != widget.fields[i].controller,
        )) {
      revision++;
      errors.clear();
      attempted = false;
      if (!uncertain) message = null;
    }
    _syncFocus();
  }

  @override
  void dispose() {
    for (final node in focus.values) {
      node.dispose();
    }
    super.dispose();
  }

  void _changed(BPTextField field, String value) {
    setState(() {
      if (!uncertain) message = null;
      if (attempted) {
        final error = field.validate?.call(value);
        if (error == null) {
          errors.remove(field.id);
        } else {
          errors[field.id] = error;
        }
      }
    });
  }

  Future<void> _submit() async {
    if (busy || uncertain) return;
    final values = Map<String, String>.unmodifiable({
      for (final f in widget.fields) f.id: f.controller.text,
    });
    final invalid = <String, String>{};
    for (final f in widget.fields) {
      final error = f.validate?.call(values[f.id]!);
      if (error != null) invalid[f.id] = error;
    }
    setState(() {
      attempted = true;
      errors
        ..clear()
        ..addAll(invalid);
      message = null;
      busy = invalid.isEmpty;
    });
    if (invalid.isNotEmpty) {
      final id = invalid.keys.first;
      focus[id]?.requestFocus();
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (!mounted) return;
        final target = focus[id]?.context;
        if (target != null) Scrollable.ensureVisible(target);
      });
      return;
    }
    FocusManager.instance.primaryFocus?.unfocus();
    final startedRevision = revision;
    bool? ok;
    try {
      ok = await widget.onSubmit(values);
    } catch (_) {
      // Never render raw transport or database exceptions.
    }
    if (!mounted) return;
    setState(() {
      busy = false;
      if (ok == null) {
        uncertain = true;
        message = "Resultado sin confirmar. Consulta el estado antes de volver a guardar.";
      } else if (startedRevision == revision) {
        message = ok
            ? widget.successMessage
            : "No se pudo guardar. Intenta nuevamente.";
      }
    });
  }

  @override
  Widget build(BuildContext context) {
    final theme = BPVisuals.of(context);
    return ColoredBox(
      color: theme.background,
      child: SafeArea(
        child: ListView(
          padding: const EdgeInsets.all(BPTokens.lg),
          children: [
            Center(
              child: ConstrainedBox(
                constraints: const BoxConstraints(
                  maxWidth: BPTokens.formMaxWidth,
                ),
                child: BPCard(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.stretch,
                    children: [
                      for (var i = 0; i < widget.fields.length; i++)
                        _field(widget.fields[i], i, theme),
                      BPButton(
                        label: busy ? "Guardando…" : "Guardar",
                        onPressed: busy || uncertain ? null : _submit,
                      ),
                      if (busy || message != null)
                        Padding(
                          padding: const EdgeInsets.only(top: BPTokens.md),
                          child: Semantics(
                            liveRegion: true,
                            child: Text(
                              busy ? "Enviando formulario" : message!,
                              style: theme.label,
                            ),
                          ),
                        ),
                    ],
                  ),
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _field(BPTextField field, int index, BPVisuals theme) {
    final last = index == widget.fields.length - 1;
    return Padding(
      key: ValueKey(field.id),
      padding: const EdgeInsets.only(bottom: BPTokens.lg),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Text(field.label, style: theme.label),
          const SizedBox(height: BPTokens.sm),
          BPInput(
            controller: field.controller,
            label: field.label,
            placeholder: field.placeholder,
            enabled: !busy && !uncertain,
            focusNode: focus[field.id],
            textInputAction: last ? TextInputAction.done : TextInputAction.next,
            onChanged: (v) => _changed(field, v),
            onSubmitted: (_) {
              if (busy || uncertain) return;
              if (last) {
                _submit();
              } else {
                focus[widget.fields[index + 1].id]?.requestFocus();
              }
            },
          ),
          if (errors.containsKey(field.id))
            Padding(
              padding: const EdgeInsets.only(top: BPTokens.xs),
              child: Semantics(
                liveRegion: true,
                child: Text(errors[field.id]!, style: theme.label),
              ),
            ),
        ],
      ),
    );
  }
}
