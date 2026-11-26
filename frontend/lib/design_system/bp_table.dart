import "package:flutter/widgets.dart";

import "design_system.dart";
import "bp_view_states.dart";

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
    required this.selectedId,
    required this.onSelect,
    required this.onPage,
    this.state = BPTableState.ready,
    this.onRetry,
  }) : columns = List.unmodifiable(columns),
       rows = List.unmodifiable(
         rows.map((r) => BPTableRow(r.id, List.unmodifiable(r.cells))),
       ) {
    if (columns.isEmpty ||
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
  final ValueChanged<String> onSelect;
  final ValueChanged<int> onPage;
  final BPTableState state;
  final VoidCallback? onRetry;

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
            return ListView(
              key: Key(wide ? "bp-table-wide" : "bp-table-compact"),
              padding: const EdgeInsets.all(BPTokens.lg),
              children: [
                Text("Buscar registros", style: theme.label),
                const SizedBox(height: BPTokens.sm),
                BPInput(
                  controller: searchController,
                  label: "Buscar registros",
                  placeholder: "Nombre o código",
                  onChanged: onSearch,
                ),
                const SizedBox(height: BPTokens.lg),
                if (state == BPTableState.loading)
                  const BPLoadingState()
                else if (state == BPTableState.error)
                  BPErrorState(onRetry: onRetry)
                else if (rows.isEmpty)
                  const BPEmptyState()
                else ...[
                  if (wide)
                    BPCard(
                      child: Row(
                        children: [
                          for (final column in columns)
                            Expanded(child: Text(column, style: theme.label)),
                          const SizedBox(width: BPTokens.sidebarWidth),
                        ],
                      ),
                    ),
                  for (final row in rows)
                    Padding(
                      padding: const EdgeInsets.only(top: BPTokens.sm),
                      child: BPCard(child: _row(row, wide, theme)),
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
    final action = Semantics(
      selected: selected,
      child: BPButton(
        label: selected
            ? "Seleccionado ${row.cells.first}"
            : "Seleccionar ${row.cells.first}",
        primary: selected,
        onPressed: () => onSelect(row.id),
      ),
    );
    if (wide) {
      return Row(
        children: [
          for (var i = 0; i < columns.length; i++)
            Expanded(
              child: Semantics(label: columns[i], child: Text(row.cells[i])),
            ),
          SizedBox(width: BPTokens.sidebarWidth, child: action),
        ],
      );
    }
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        for (var i = 0; i < columns.length; i++)
          Padding(
            padding: const EdgeInsets.only(bottom: BPTokens.sm),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(columns[i], style: theme.label),
                Text(row.cells[i]),
              ],
            ),
          ),
        action,
      ],
    );
  }
}
