import "package:flutter/widgets.dart";

import "../design_system/bp_badge.dart";
import "../design_system/bp_choice_bar.dart";
import "../design_system/bp_table.dart";
import "../design_system/design_system.dart";

/// Read-only projection of a fixture. No stock calculation is performed here.
class PreviewProduct {
  const PreviewProduct(this.id, this.name, this.code, this.stock);
  final String id, name, code;
  final int stock;
  String get status => stock == 0
      ? "Agotado"
      : stock < 10
      ? "Stock bajo"
      : "Disponible";
  BPTone get tone => stock == 0
      ? BPTone.danger
      : stock < 10
      ? BPTone.warning
      : BPTone.success;
}

class ProductDetail extends StatelessWidget {
  const ProductDetail({
    super.key,
    required this.product,
    required this.onBack,
    required this.onEdit,
    required this.onHistory,
  });
  final PreviewProduct product;
  final VoidCallback onBack, onEdit, onHistory;

  @override
  Widget build(BuildContext context) {
    final theme = BPVisuals.of(context);
    return ListView(
      key: const ValueKey("product-detail"),
      padding: const EdgeInsets.all(BPTokens.lg),
      children: [
        Text("Ficha del producto", style: theme.heading),
        const SizedBox(height: BPTokens.sm),
        Text("Consulta ficticia · Cambios temporales", style: theme.muted),
        const SizedBox(height: BPTokens.lg),
        BPCard(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              Text(product.name, style: theme.heading),
              const SizedBox(height: BPTokens.sm),
              Text("Código", style: theme.muted),
              Text(product.code, style: theme.label),
              const SizedBox(height: BPTokens.lg),
              Text("Existencias de ejemplo", style: theme.muted),
              Text("${product.stock} und.", style: theme.heading),
              const SizedBox(height: BPTokens.sm),
              Align(
                alignment: Alignment.centerLeft,
                child: BPBadge(label: product.status, tone: product.tone),
              ),
              const SizedBox(height: BPTokens.lg),
              BPButton(label: "Editar producto", onPressed: onEdit),
              const SizedBox(height: BPTokens.sm),
              BPButton(
                label: "Consultar movimientos",
                primary: false,
                onPressed: onHistory,
              ),
            ],
          ),
        ),
        const SizedBox(height: BPTokens.lg),
        BPButton(
          label: "Volver a productos",
          primary: false,
          onPressed: onBack,
        ),
      ],
    );
  }
}

class _Movement {
  const _Movement(
    this.id,
    this.productId,
    this.kind,
    this.quantity,
    this.date,
    this.reference,
  );
  final String id, productId, kind, quantity, date, reference;
  String get label => kind == "in" ? "Entrada de ejemplo" : "Salida de ejemplo";
}

// Static presentation fixtures, not a ledger or a source of stock authority.
const _movements = [
  _Movement("m6", "3", "out", "5", "24/11/2026", "DEMO-006"),
  _Movement("m4", "2", "out", "4", "24/11/2026", "DEMO-004"),
  _Movement("m2", "1", "out", "2", "24/11/2026", "DEMO-002"),
  _Movement("m5", "3", "in", "5", "23/11/2026", "DEMO-005"),
  _Movement("m3", "2", "in", "10", "23/11/2026", "DEMO-003"),
  _Movement("m1", "1", "in", "50", "23/11/2026", "DEMO-001"),
];

class MovementHistory extends StatefulWidget {
  const MovementHistory({
    super.key,
    required this.products,
    this.productId,
    this.onBack,
  });
  final List<PreviewProduct> products;
  final String? productId;
  final VoidCallback? onBack;

  @override
  State<MovementHistory> createState() => _MovementHistoryState();
}

class _MovementHistoryState extends State<MovementHistory> {
  final search = TextEditingController();
  String kind = "all";
  _Movement? detail;

  @override
  void dispose() {
    search.dispose();
    super.dispose();
  }

  PreviewProduct _product(String id) =>
      widget.products.firstWhere((p) => p.id == id);

  @override
  Widget build(BuildContext context) {
    final theme = BPVisuals.of(context);
    final current = detail;
    if (current != null) {
      final product = _product(current.productId);
      return ListView(
        padding: const EdgeInsets.all(BPTokens.lg),
        children: [
          Text("Detalle del movimiento", style: theme.heading),
          const SizedBox(height: BPTokens.sm),
          Text("Registro ficticio · Solo consulta", style: theme.muted),
          const SizedBox(height: BPTokens.lg),
          BPCard(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                Text(current.label, style: theme.title),
                const SizedBox(height: BPTokens.md),
                for (final entry in [
                  ("Producto", product.name),
                  ("Código actual", product.code),
                  ("Cantidad", "${current.quantity} und."),
                  ("Fecha de ejemplo", current.date),
                  ("Referencia ficticia", current.reference),
                ]) ...[
                  Text(entry.$1, style: theme.muted),
                  Text(entry.$2, style: theme.label),
                  const SizedBox(height: BPTokens.sm),
                ],
              ],
            ),
          ),
          const SizedBox(height: BPTokens.lg),
          BPButton(
            label: "Volver al historial",
            primary: false,
            onPressed: () => setState(() => detail = null),
          ),
        ],
      );
    }
    final query = search.text.trim().toLowerCase();
    final visible = _movements.where((m) {
      final product = _product(m.productId);
      return (widget.productId == null || widget.productId == m.productId) &&
          (kind == "all" || kind == m.kind) &&
          [
            product.name,
            product.code,
            m.reference,
          ].any((v) => v.toLowerCase().contains(query));
    }).toList();
    return BPTable(
      key: const ValueKey("movement-history"),
      columns: const ["Producto", "Referencia", "Tipo", "Cantidad", "Fecha"],
      rows: [
        for (final m in visible)
          BPTableRow(m.id, [
            _product(m.productId).name,
            m.reference,
            m.label,
            m.quantity,
            m.date,
          ]),
      ],
      page: 0,
      pageCount: visible.isEmpty ? 0 : 1,
      searchController: search,
      onSearch: (_) => setState(() {}),
      onPage: (_) {},
      rowActionLabel: "Consultar",
      onRowAction: (id) =>
          setState(() => detail = visible.firstWhere((m) => m.id == id)),
      cellBuilder: (row, index) => index == 2
          ? BPBadge(
              label: row.cells[index],
              tone: row.cells[index] == "Entrada de ejemplo"
                  ? BPTone.success
                  : BPTone.neutral,
            )
          : null,
      clearSearchLabel: "Limpiar filtros",
      onClearSearch: () => setState(() {
        search.clear();
        kind = "all";
      }),
      toolbar: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text("Historial de ejemplo", style: theme.heading),
          const SizedBox(height: BPTokens.sm),
          Text(
            "Los movimientos reales requieren la API protegida. Estos registros son ficticios.",
            style: theme.muted,
          ),
          if (widget.productId != null) ...[
            const SizedBox(height: BPTokens.sm),
            Text(_product(widget.productId!).name, style: theme.title),
            const SizedBox(height: BPTokens.sm),
            BPButton(
              label: "Volver al producto",
              primary: false,
              onPressed: widget.onBack,
            ),
          ],
        ],
      ),
      filters: BPChoiceBar(
        label: "Tipo de movimiento",
        keyPrefix: "movement-filter",
        choices: const [
          BPChoice("all", "Todos"),
          BPChoice("in", "Entradas"),
          BPChoice("out", "Salidas"),
        ],
        selectedId: kind,
        onChanged: (id) => setState(() => kind = id),
      ),
    );
  }
}
