import "package:flutter/widgets.dart";

import "design_system.dart";
import "bp_view_states.dart";

import "package:shadcn_ui/shadcn_ui.dart";

enum BPTableState { ready, loading, error }

class BPTableRow {
  const BPTableRow(this.id, this.cells);
  final String id;
  final List<String> cells;
}

/// Displays one supplied page. The caller owns filtering, selection,
/// pagination and asynchronous requests. Never fetches or mutates records.
class BPTable extends StatelessWidget {
  BPTable({
    super.key,
    required List<String> columns,
    required List<BPTableRow> rows,
    required this.page,
    required this.pageCount,
    required this.searchController,
    required this.onSearch,
    this.selectedId,
    this.onSelect,
    this.onRowAction,
    this.onRowDetails,
    this.rowActionLabel = "Abrir",
    required this.onPage,
    this.state = BPTableState.ready,
    this.onRetry,
    this.toolbar,
    this.filters,
    this.clearSearchLabel = "Limpiar búsqueda",
    this.onClearSearch,
    this.cellBuilder,
  }) : columns = List.unmodifiable(columns),
       rows = List.unmodifiable(
         rows.map((r) => BPTableRow(r.id, List.unmodifiable(r.cells))),
       ) {
    if ((onSelect == null) == (onRowAction == null) ||
        (onRowAction != null && selectedId != null) ||
        rowActionLabel.trim().isEmpty ||
        columns.isEmpty ||
        columns.length > 8 ||
        columns.any((c) => c.trim().isEmpty) ||
        rows.length > 50 ||
        pageCount < 0 ||
        page < 0 ||
        (pageCount == 0 ? page != 0 : page >= pageCount) ||
        (state == BPTableState.ready && pageCount == 0 && rows.isNotEmpty) ||
        rows.any((r) => r.id.isEmpty || r.cells.length != columns.length) ||
        rows.map((r) => r.id).toSet().length != rows.length) {
      throw ArgumentError("Invalid BPTable page");
    }
  }
  final List<String> columns;
  final List<BPTableRow> rows;
  final int page, pageCount;
  final TextEditingController searchController;
  final ValueChanged<String> onSearch;
  final String? selectedId;
  final ValueChanged<String>? onSelect, onRowAction, onRowDetails;
  final String rowActionLabel, clearSearchLabel;
  final ValueChanged<int> onPage;
  final BPTableState state;
  final VoidCallback? onRetry, onClearSearch;
  final Widget? toolbar, filters;
  final Widget? Function(BPTableRow, int)? cellBuilder;

  static const double tableMinWidth = 840, columnMinWidth = 116;
  double get actionWidth => onRowAction == null
      ? (onRowDetails == null ? 176 : 224)
      : BPTokens.touchTarget * (onRowDetails == null ? 1 : 2) + BPTokens.sm;

  @override
  Widget build(BuildContext context) {
    final theme = BPVisuals.of(context);
    return ColoredBox(
      color: theme.background,
      child: SafeArea(
        child: LayoutBuilder(
          builder: (context, constraints) {
            final wide =
                constraints.maxWidth >= tableMinWidth &&
                constraints.maxWidth >=
                    columns.length * columnMinWidth +
                        actionWidth +
                        BPTokens.xxl * 2 &&
                MediaQuery.textScalerOf(context).scale(14) <= 21;
            final search = Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text("Buscar registros", style: theme.label),
                const SizedBox(height: BPTokens.xs),
                BPInput(
                  controller: searchController,
                  label: "Buscar registros",
                  placeholder: "Nombre o código",
                  onChanged: onSearch,
                ),
              ],
            );
            return ListView(
              key: Key(wide ? "bp-table-wide" : "bp-table-compact"),
              padding: const EdgeInsets.all(BPTokens.lg),
              children: [
                if (toolbar != null) ...[
                  toolbar!,
                  const SizedBox(height: BPTokens.lg),
                ],
                if (wide && filters != null)
                  Row(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Expanded(flex: 2, child: search),
                      const SizedBox(width: BPTokens.lg),
                      Expanded(flex: 3, child: filters!),
                    ],
                  )
                else ...[
                  search,
                  if (filters != null) ...[
                    const SizedBox(height: BPTokens.md),
                    filters!,
                  ],
                ],
                const SizedBox(height: BPTokens.md),
                if (state == BPTableState.loading)
                  const BPLoadingState()
                else if (state == BPTableState.error)
                  BPErrorState(onRetry: onRetry)
                else if (rows.isEmpty)
                  BPEmptyState(
                    actionLabel: clearSearchLabel,
                    onAction: onClearSearch,
                  )
                else ...[
                  if (wide)
                    Container(
                      key: const ValueKey("bp-data-surface"),
                      decoration: BoxDecoration(
                        color: theme.surface,
                        border: Border.all(color: theme.border),
                        borderRadius: BorderRadius.circular(BPTokens.radius),
                      ),
                      clipBehavior: Clip.antiAlias,
                      child: Column(
                        children: [
                          ColoredBox(
                            color: theme.background,
                            child: Padding(
                              padding: const EdgeInsets.all(BPTokens.md),
                              child: Row(
                                children: [
                                  for (var i = 0; i < columns.length; i++)
                                    Expanded(
                                      flex: i == 0 ? 2 : 1,
                                      child: Padding(
                                        padding: const EdgeInsets.only(
                                          right: BPTokens.sm,
                                        ),
                                        child: Text(
                                          columns[i],
                                          style: theme.label,
                                        ),
                                      ),
                                    ),
                                  SizedBox(
                                    width: actionWidth,
                                    child: Text("Acciones", style: theme.label),
                                  ),
                                ],
                              ),
                            ),
                          ),
                          for (final row in rows)
                            Container(
                              key: ValueKey("bp-data-row-${row.id}"),
                              decoration: BoxDecoration(
                                border: Border(
                                  top: BorderSide(color: theme.border),
                                ),
                              ),
                              padding: const EdgeInsets.symmetric(
                                horizontal: BPTokens.md,
                                vertical: BPTokens.xs,
                              ),
                              child: _row(row, true, theme),
                            ),
                        ],
                      ),
                    )
                  else
                    for (final row in rows)
                      Padding(
                        padding: const EdgeInsets.only(top: BPTokens.xs),
                        child: BPCard(
                          compact: true,
                          child: _row(row, false, theme),
                        ),
                      ),
                ],
                if (state == BPTableState.ready && pageCount > 0) ...[
                  const SizedBox(height: BPTokens.lg),
                  Semantics(
                    liveRegion: true,
                    child: Text(
                      "Página ${page + 1} de $pageCount",
                      style: theme.muted,
                    ),
                  ),
                  const SizedBox(height: BPTokens.sm),
                  Wrap(
                    spacing: BPTokens.sm,
                    runSpacing: BPTokens.sm,
                    children: [
                      BPButton(
                        label: "Anterior",
                        primary: false,
                        onPressed: page > 0 ? () => onPage(page - 1) : null,
                      ),
                      BPButton(
                        label: "Siguiente",
                        primary: false,
                        onPressed: page + 1 < pageCount
                            ? () => onPage(page + 1)
                            : null,
                      ),
                    ],
                  ),
                ],
              ],
            );
          },
        ),
      ),
    );
  }

  Widget _row(BPTableRow row, bool wide, BPVisuals theme) {
    final selected = row.id == selectedId;
    final action = onRowAction != null
        ? _RowAction(
            key: ValueKey("bp-row-action-${row.id}"),
            label: rowActionLabel,
            contextLabel: row.cells.first,
            wide: wide,
            icon: rowActionLabel == "Editar"
                ? LucideIcons.pencil
                : LucideIcons.arrowRight,
            onPressed: () => onRowAction!(row.id),
          )
        : Semantics(
            selected: selected,
            child: BPButton(
              label: selected
                  ? "Seleccionado ${row.cells.first}"
                  : "Seleccionar ${row.cells.first}",
              primary: selected,
              onPressed: () => onSelect!(row.id),
            ),
          );
    final actions = onRowDetails == null
        ? action
        : Wrap(
            spacing: BPTokens.xs,
            runSpacing: BPTokens.xs,
            children: [
              _RowAction(
                key: ValueKey("bp-row-details-${row.id}"),
                label: "Ver",
                contextLabel: row.cells.first,
                wide: wide,
                icon: LucideIcons.eye,
                onPressed: () => onRowDetails!(row.id),
              ),
              action,
            ],
          );
    if (wide) {
      return Row(
        children: [
          for (var i = 0; i < columns.length; i++)
            Expanded(
              flex: i == 0 ? 2 : 1,
              child: Padding(
                padding: const EdgeInsets.only(right: BPTokens.sm),
                child: Semantics(
                  label: columns[i],
                  child: Align(
                    alignment: Alignment.centerLeft,
                    child: _cell(row, i),
                  ),
                ),
              ),
            ),
          SizedBox(width: actionWidth, child: actions),
        ],
      );
    }
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Semantics(
          label: columns.first,
          child: Text(row.cells.first, style: theme.title),
        ),
        const SizedBox(height: BPTokens.sm),
        for (var i = 1; i < columns.length; i += 2)
          Padding(
            padding: const EdgeInsets.only(bottom: BPTokens.sm),
            child: Row(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Expanded(child: _metadata(row, i, theme)),
                const SizedBox(width: BPTokens.sm),
                Expanded(
                  child: i + 1 < columns.length
                      ? _metadata(row, i + 1, theme)
                      : const SizedBox.shrink(),
                ),
              ],
            ),
          ),
        if (onRowAction != null)
          Align(alignment: Alignment.centerRight, child: actions)
        else
          actions,
      ],
    );
  }

  Widget _cell(BPTableRow row, int index) =>
      cellBuilder?.call(row, index) ?? Text(row.cells[index]);

  Widget _metadata(BPTableRow row, int index, BPVisuals theme) => Column(
    crossAxisAlignment: CrossAxisAlignment.start,
    children: [
      Text(columns[index], style: theme.muted),
      _cell(row, index),
    ],
  );
}

/// Compact actions retain full accessible names; touch layouts show labels.
class _RowAction extends StatelessWidget {
  const _RowAction({
    super.key,
    required this.label,
    required this.contextLabel,
    required this.wide,
    required this.icon,
    required this.onPressed,
  });
  final String label, contextLabel;
  final bool wide;
  final IconData icon;
  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) {
    final accessible = "$label $contextLabel";
    final button = ConstrainedBox(
      constraints: const BoxConstraints(
        minHeight: BPTokens.touchTarget,
        minWidth: BPTokens.touchTarget,
      ),
      child: ShadButton.raw(
        variant: wide ? ShadButtonVariant.ghost : ShadButtonVariant.outline,
        height: 0,
        padding: const EdgeInsets.all(BPTokens.sm),
        onPressed: onPressed,
        child: Semantics(
          label: accessible,
          child: ExcludeSemantics(
            child: wide
                ? Icon(icon, size: 18)
                : Text(label, textAlign: TextAlign.center),
          ),
        ),
      ),
    );
    return wide
        ? ShadTooltip(
            builder: (_) => Text(accessible),
            effects: const [],
            child: SizedBox(
              width: BPTokens.touchTarget,
              height: BPTokens.touchTarget,
              child: button,
            ),
          )
        : button;
  }
}

/// Responsive title and operation slot; the caller owns all actions.
class BPTableHeader extends StatelessWidget {
  const BPTableHeader({
    super.key,
    required this.title,
    required this.description,
    required this.action,
    this.feedback,
  });
  final String title, description;
  final Widget action;
  final Widget? feedback;
  @override
  Widget build(BuildContext context) {
    final theme = BPVisuals.of(context);
    final heading = Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(title, style: theme.heading),
        const SizedBox(height: BPTokens.xs),
        Text(description, style: theme.muted),
      ],
    );
    return LayoutBuilder(
      builder: (context, constraints) {
        final horizontal =
            constraints.maxWidth >= BPTable.tableMinWidth - BPTokens.xxl &&
            MediaQuery.textScalerOf(context).scale(14) <= 21;
        return Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            if (horizontal)
              Row(
                crossAxisAlignment: CrossAxisAlignment.center,
                children: [
                  Expanded(child: heading),
                  const SizedBox(width: BPTokens.md),
                  action,
                ],
              )
            else ...[
              heading,
              const SizedBox(height: BPTokens.md),
              action,
            ],
            if (feedback != null) ...[
              const SizedBox(height: BPTokens.sm),
              feedback!,
            ],
          ],
        );
      },
    );
  }
}
