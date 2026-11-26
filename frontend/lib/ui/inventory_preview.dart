import "package:flutter/widgets.dart";

import "../design_system/design_system.dart";
import "../design_system/bp_page_layout.dart";
import "../design_system/bp_table.dart";
import "../design_system/bp_form.dart";
import "../design_system/bp_badge.dart";
import "../design_system/bp_choice_bar.dart";
import "inventory_consultation.dart";
import "inventory_dashboard.dart";

class PreviewApp extends StatelessWidget {
  const PreviewApp({super.key});
  @override
  Widget build(BuildContext context) => const BPApp(home: _InventoryPreview());
}

class _InventoryPreview extends StatefulWidget {
  const _InventoryPreview();
  @override
  State<_InventoryPreview> createState() => _InventoryPreviewState();
}

class _InventoryPreviewState extends State<_InventoryPreview> {
  final products = <PreviewProduct>[
    const PreviewProduct("1", "Arroz 1 kg", "ALI-001", 48),
    const PreviewProduct("2", "Café 250 g", "ALI-002", 6),
    const PreviewProduct("3", "Aceite 1 L", "ALI-003", 0),
  ];
  final search = TextEditingController();
  final name = TextEditingController();
  final code = TextEditingController();
  static const pageSize = 10;
  String selected = "products";
  String? editingId, feedback, detailId, historyProductId;
  String filter = "all";
  bool editing = false;
  int page = 0, nextId = 4;

  @override
  void dispose() {
    search.dispose();
    name.dispose();
    code.dispose();
    super.dispose();
  }

  void _startEdit(PreviewProduct? item) {
    setState(() {
      editing = true;
      feedback = null;
      editingId = item?.id;
      name.text = item?.name ?? "";
      code.text = item?.code ?? "";
    });
  }

  String? _validateName(String value) {
    if (value.trim().isEmpty) return "Escribe un nombre";
    if (value.trim().length > 100) return "Máximo 100 caracteres";
    return null;
  }

  String? _validateCode(String value) {
    final normalized = value.trim().toUpperCase();
    if (normalized.isEmpty) return "Escribe un código";
    if (normalized.length > 40) return "Máximo 40 caracteres";
    if (products.any(
      (p) => p.id != editingId && p.code.toUpperCase() == normalized,
    )) {
      return "Ese código ya existe en el ejemplo";
    }
    return null;
  }

  Future<bool> _save(Map<String, String> values) async {
    final newName = values["name"]!, newCode = values["code"]!;
    if (!editing ||
        _validateName(newName) != null ||
        _validateCode(newCode) != null)
      return false;
    setState(() {
      final index = products.indexWhere((p) => p.id == editingId);
      if (index < 0) {
        products.add(
          PreviewProduct(
            (nextId++).toString(),
            newName.trim(),
            newCode.trim().toUpperCase(),
            0,
          ),
        );
      } else {
        final old = products[index];
        products[index] = PreviewProduct(
          old.id,
          newName.trim(),
          newCode.trim().toUpperCase(),
          old.stock,
        );
      }
      editing = false;
      editingId = null;
      detailId = null;
      feedback = index < 0
          ? "Producto creado en esta sesión"
          : "Producto actualizado en esta sesión";
      filter = "all";
      search.text = newCode.trim();
      page = 0;
    });
    return true;
  }

  @override
  Widget build(BuildContext context) {
    return BPPageLayout(
      title: "BalanzaPro3.0",
      contextLabel: "Entorno de desarrollo · Datos ficticios",
      items: const [
        BPNavItem("home", "Inicio", icon: BPNavIcon.home),
        BPNavItem("products", "Productos", icon: BPNavIcon.products),
        BPNavItem("movements", "Movimientos", icon: BPNavIcon.movements),
      ],
      selectedId: selected,
      onNavigate: (id) => setState(() {
        selected = id;
        if (id == "movements") historyProductId = null;
      }),
      child: LayoutBuilder(
        builder: (context, constraints) {
          final theme = BPVisuals.of(context);
          if (selected == "home") {
            return InventoryDashboard(
              products: products,
              hasDraft: editing,
              onFilter: (id) => _setFilter(id, fromDashboard: true),
              onProducts: () => setState(() => selected = "products"),
              onNew: () {
                if (editing) {
                  setState(() => selected = "products");
                } else {
                  setState(() {
                    selected = "products";
                    detailId = null;
                  });
                  _startEdit(null);
                }
              },
              onMovements: () => setState(() {
                historyProductId = null;
                selected = "movements";
              }),
            );
          }
          if (selected == "movements") {
            return MovementHistory(
              key: ValueKey("history-$historyProductId"),
              products: products,
              productId: historyProductId,
              onBack: () => setState(() => selected = "products"),
            );
          }
          if (editing) return _editor(theme, constraints);
          if (detailId != null) {
            final product = products.firstWhere((p) => p.id == detailId);
            return ProductDetail(
              product: product,
              onBack: () => setState(() => detailId = null),
              onEdit: () => _startEdit(product),
              onHistory: () => setState(() {
                historyProductId = product.id;
                selected = "movements";
              }),
            );
          }
          return _products(theme);
        },
      ),
    );
  }

  bool _matchesFilter(PreviewProduct p) => switch (filter) {
    "available" => p.stock >= 10,
    "low" => p.stock > 0 && p.stock < 10,
    "empty" => p.stock == 0,
    _ => true,
  };

  void _setFilter(String id, {bool fromDashboard = false}) {
    setState(() {
      filter = id;
      page = 0;
      feedback = null;
      if (fromDashboard) {
        search.clear();
        selected = "products";
        detailId = null;
      }
    });
  }

  Widget _products(BPVisuals theme) {
    final query = search.text.trim().toLowerCase();
    final visible = products
        .where(
          (p) =>
              _matchesFilter(p) &&
              (p.name.toLowerCase().contains(query) ||
                  p.code.toLowerCase().contains(query)),
        )
        .toList();
    final pages = (visible.length + pageSize - 1) ~/ pageSize;
    return BPTable(
      key: const ValueKey("products"),
      columns: const ["Producto", "Código", "Stock", "Estado"],
      rows: [
        for (final p in visible.skip(page * pageSize).take(pageSize))
          BPTableRow(p.id, [p.name, p.code, p.stock.toString(), p.status]),
      ],
      cellBuilder: (row, index) {
        if (index != 3) return null;
        final label = row.cells[index];
        return BPBadge(
          key: ValueKey("bp-status-${row.id}"),
          label: label,
          tone: switch (label) {
            "Disponible" => BPTone.success,
            "Stock bajo" => BPTone.warning,
            "Agotado" => BPTone.danger,
            _ => BPTone.neutral,
          },
        );
      },
      page: page,
      pageCount: pages,
      searchController: search,
      rowActionLabel: "Editar",
      onRowDetails: (id) => setState(() => detailId = id),
      onRowAction: (id) => _startEdit(products.firstWhere((p) => p.id == id)),
      onPage: (index) => setState(() => page = index),
      onSearch: (_) => setState(() {
        page = 0;
        feedback = null;
      }),
      clearSearchLabel: "Limpiar filtros",
      onClearSearch: () => setState(() {
        search.clear();
        filter = "all";
        page = 0;
        feedback = null;
      }),
      filters: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          BPChoiceBar(
            label: "Filtrar por existencias",
            keyPrefix: "product-filter",
            choices: const [
              BPChoice("all", "Todos"),
              BPChoice("available", "Disponibles"),
              BPChoice("low", "Bajo stock"),
              BPChoice("empty", "Agotados"),
            ],
            selectedId: filter,
            onChanged: _setFilter,
          ),
          const SizedBox(height: BPTokens.md),
          Semantics(
            liveRegion: true,
            child: Text(
              "${visible.length} de ${products.length} productos de ejemplo",
              style: theme.muted,
            ),
          ),
        ],
      ),
      toolbar: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text("Productos", style: theme.heading),
          const SizedBox(height: BPTokens.xs),
          Text("Prueba local · Cambios temporales", style: theme.muted),
          const SizedBox(height: BPTokens.md),
          BPButton(label: "Nuevo producto", onPressed: () => _startEdit(null)),
          if (feedback != null) ...[
            const SizedBox(height: BPTokens.sm),
            Semantics(
              liveRegion: true,
              child: BPBadge(label: feedback!, tone: BPTone.success),
            ),
          ],
        ],
      ),
    );
  }

  Widget _editor(BPVisuals theme, BoxConstraints constraints) => Column(
    children: [
      ConstrainedBox(
        constraints: BoxConstraints(maxHeight: constraints.maxHeight / 2),
        child: SingleChildScrollView(
          child: Padding(
            padding: const EdgeInsets.all(BPTokens.lg),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                Text(
                  editingId == null
                      ? "Nuevo producto de ejemplo"
                      : "Editar producto de ejemplo",
                  style: theme.heading,
                ),
                const SizedBox(height: BPTokens.sm),
                Text(
                  "Solo memoria local; no se guardan datos reales.",
                  style: theme.muted,
                ),
                const SizedBox(height: BPTokens.sm),
                BPButton(
                  label: "Cancelar borrador",
                  primary: false,
                  onPressed: () => setState(() {
                    editing = false;
                    editingId = null;
                  }),
                ),
              ],
            ),
          ),
        ),
      ),
      Expanded(
        child: BPForm(
          fields: [
            BPTextField(
              id: "name",
              label: "Nombre del producto",
              controller: name,
              validate: _validateName,
            ),
            BPTextField(
              id: "code",
              label: "Código",
              controller: code,
              validate: _validateCode,
            ),
          ],
          onSubmit: _save,
        ),
      ),
    ],
  );
}
