import "package:flutter/widgets.dart";

import "design_system.dart";

/// A confirmed empty result, not an error or an unconfirmed load.
class BPEmptyState extends StatelessWidget {
  const BPEmptyState({
    super.key,
    this.title = "No hay resultados",
    this.description,
    this.actionLabel = "Limpiar búsqueda",
    this.onAction,
  });
  final String title, actionLabel;
  final String? description;
  final VoidCallback? onAction;

  @override
  Widget build(BuildContext context) => _BPStateCard(
    title: title,
    description: description,
    actionLabel: actionLabel,
    onAction: onAction,
  );
}

/// Only for recoverable reads or confirmed rejections. Unknown outcomes
/// require a separate reconciliation flow, not a retry of a write.
class BPErrorState extends StatelessWidget {
  const BPErrorState({
    super.key,
    this.title = "No se pudieron cargar los registros",
    this.description,
    this.onRetry,
  });
  final String title;
  final String? description;
  final VoidCallback? onRetry;
  @override
  Widget build(BuildContext context) => _BPStateCard(
    title: title,
    description: description,
    actionLabel: "Reintentar",
    onAction: onRetry,
  );
}

/// Presentation only. The caller determines connectivity and data freshness.
/// The action checks connectivity; it must not replay pending writes.
class BPOfflineState extends StatelessWidget {
  const BPOfflineState({super.key, this.onRetry});
  final VoidCallback? onRetry;
  @override
  Widget build(BuildContext context) => _BPStateCard(
    title: "Sin conexión",
    description: "Comprueba el acceso al servidor antes de continuar.",
    actionLabel: "Comprobar conexión",
    onAction: onRetry,
  );
}

/// A caller-driven loading region. Static default skeletons for short lists;
/// callers can supply their own geometry using the shared BPSkeleton.
class BPLoadingState extends StatelessWidget {
  const BPLoadingState({
    super.key,
    this.label = "Cargando registros",
    this.placeholder,
  });
  final String label;
  final Widget? placeholder;
  @override
  Widget build(BuildContext context) {
    final theme = BPVisuals.of(context);
    return BPCard(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Semantics(liveRegion: true, child: Text(label, style: theme.label)),
          const SizedBox(height: BPTokens.sm),
          ExcludeSemantics(
            child:
                placeholder ??
                const Column(
                  children: [
                    BPSkeleton(
                      width: double.infinity,
                      height: BPTokens.touchTarget,
                    ),
                    SizedBox(height: BPTokens.sm),
                    BPSkeleton(
                      width: double.infinity,
                      height: BPTokens.touchTarget,
                    ),
                  ],
                ),
          ),
        ],
      ),
    );
  }
}

class _BPStateCard extends StatelessWidget {
  const _BPStateCard({
    required this.title,
    this.description,
    required this.actionLabel,
    this.onAction,
  });
  final String title, actionLabel;
  final String? description;
  final VoidCallback? onAction;
  @override
  Widget build(BuildContext context) {
    final theme = BPVisuals.of(context);
    return BPCard(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Semantics(liveRegion: true, child: Text(title, style: theme.title)),
          if (description != null) ...[
            const SizedBox(height: BPTokens.sm),
            Text(description!, style: theme.muted),
          ],
          if (onAction != null) ...[
            const SizedBox(height: BPTokens.md),
            BPButton(label: actionLabel, onPressed: onAction, primary: false),
          ],
        ],
      ),
    );
  }
}
