import "package:flutter/widgets.dart";

import "../design_system/design_system.dart";
import "../design_system/bp_page_layout.dart";
import "../design_system/bp_table.dart";
import "../design_system/bp_form.dart";
import "../design_system/bp_badge.dart";
import "../design_system/bp_choice_bar.dart";

class PreviewApp extends StatelessWidget {
  const PreviewApp({super.key});
  @override
  Widget build(BuildContext context) => const BPApp(home: _InventoryPreview());
}

/// Synthetic records, retained only for this application instance.
class _Fixture {
  const _Fixture(this.id, this.name, this.code, this.stock);
  final String id, name, code;
  final int stock;
  String get status => stock == 0
      ? "Agotado"
      : stock < 10
      ? "Stock bajo"
      : "Disponible";
}

class _InventoryPreview extends StatefulWidget {
  const _InventoryPreview();
  @override
  State<_InventoryPreview> createState() => _InventoryPreviewState();
}

class _InventoryPreviewState extends State<_InventoryPreview> {
  final products = <_Fixture>[
    const _Fixture("1", "Arroz 1 kg", "ALI-001", 48),
    const _Fixture("2", "Café 250 g", "ALI-002", 6),
    const _Fixture("3", "Aceite 1 L", "ALI-003", 0),
  ];
  final search = TextEditingController();
  final name = TextEditingController();
  final code = TextEditingController();
  static const pageSize = 10;
  String selected = "products";
  String? editingId, feedback;
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

  void _startEdit(_Fixture? item) {
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
          _Fixture(
            (nextId++).toString(),
            newName.trim(),
            newCode.trim().toUpperCase(),
            0,
          ),
        );
      } else {
        final old = products[index];
        products[index] = _Fixture(
          old.id,
          newName.trim(),
          newCode.trim().toUpperCase(),
          old.stock,
        );
      }
      editing = false;
      editingId = null;
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
      onNavigate: (id) => setState(() => selected = id),
      child: LayoutBuilder(
        builder: (context, constraints) {
          final theme = BPVisuals.of(context);
          if (selected == "home") {
            return _dashboard(
              theme,
              constraints.maxWidth >= BPTokens.wideBreakpoint &&
                  MediaQuery.textScalerOf(context).scale(14) <= 21,
            );
          }
          if (selected == "movements") {
            return const ListViewPlaceholder();
          }
          if (editing) return _editor(theme, constraints);
          return _products(theme);
        },
      ),
    );
  }

  bool _matchesFilter(_Fixture p) => switch (filter) {
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

  Widget _dashboard(BPVisuals theme, bool wide) {
    final cards = [
      for (final metric in [
        ("Productos de ejemplo", products.length, "all", "Ver todos"),
        (
          "Stock bajo de ejemplo",
          products.where((p) => p.stock > 0 && p.stock < 10).length,
          "low",
          "Ver stock bajo",
        ),
        (
          "Agotados de ejemplo",
          products.where((p) => p.stock == 0).length,
          "empty",
          "Ver agotados",
        ),
      ])
        BPCard(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              Text(metric.$1, style: theme.label),
              const SizedBox(height: BPTokens.sm),
              Text(metric.$2.toString(), style: theme.heading),
              const SizedBox(height: BPTokens.sm),
              BPButton(
                key: ValueKey("dashboard-${metric.$3}"),
                label: metric.$4,
                primary: false,
                onPressed: () => _setFilter(metric.$3, fromDashboard: true),
              ),
            ],
          ),
        ),
    ];
    return ListView(
      key: const ValueKey("home"),
      padding: const EdgeInsets.all(BPTokens.lg),
      children: [
        Text("Resumen de ejemplo", style: theme.heading),
        const SizedBox(height: BPTokens.sm),
        Text(
          "Indicadores ficticios; no consultan el servidor.",
          style: theme.muted,
        ),
        const SizedBox(height: BPTokens.xl),
        if (wide)
          Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              for (final card in cards)
                Expanded(
                  child: Padding(
                    padding: const EdgeInsets.only(right: BPTokens.sm),
                    child: card,
                  ),
                ),
            ],
          )
        else
          for (final card in cards)
            Padding(
              padding: const EdgeInsets.only(bottom: BPTokens.sm),
              child: card,
            ),
        const SizedBox(height: BPTokens.lg),
        BPButton(
          label: "Ver productos",
          onPressed: () => setState(() => selected = "products"),
        ),
      ],
    );
  }
}

class ListViewPlaceholder extends StatelessWidget {
  const ListViewPlaceholder({super.key});
  @override
  Widget build(BuildContext context) => ListView(
    key: const ValueKey("movements"),
    padding: const EdgeInsets.all(BPTokens.lg),
    children: [
      Text("Movimientos", style: BPVisuals.of(context).heading),
      const SizedBox(height: BPTokens.lg),
      const BPCard(
        child: Text(
          "Los movimientos estarán disponibles al conectar la API protegida.",
        ),
      ),
    ],
  );
}
