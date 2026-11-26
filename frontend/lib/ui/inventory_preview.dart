import "package:flutter/widgets.dart";

import "../design_system/design_system.dart";
import "../design_system/bp_page_layout.dart";

/// Experimental UI adapter, not the final design system.
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
  static const products = [
    ("Arroz 1 kg", "ALI-001", "48", "Disponible"),
    ("Café 250 g", "ALI-002", "6", "Stock bajo"),
    ("Aceite 1 L", "ALI-003", "0", "Agotado"),
  ];
  final search = TextEditingController();
  String query = "";
  String selected = "products";
  @override
  void dispose() {
    search.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final theme = BPVisuals.of(context);
    final visible = products
        .where(
          (p) =>
              p.$1.toLowerCase().contains(query) ||
              p.$2.toLowerCase().contains(query),
        )
        .toList();
    return BPPageLayout(
      title: "BalanzaPro3.0",
      contextLabel: "Entorno de desarrollo · Datos ficticios",
      items: const [
        BPNavItem("home", "Inicio"),
        BPNavItem("products", "Productos"),
        BPNavItem("movements", "Movimientos"),
      ],
      selectedId: selected,
      onNavigate: (id) => setState(() => selected = id),
      child: LayoutBuilder(
        builder: (context, constraints) {
          final wide =
              constraints.maxWidth >= BPTokens.wideBreakpoint &&
              MediaQuery.textScalerOf(context).scale(14) <= 21;
          if (selected == "home") return _dashboard(theme, wide);
          return ListView(
            key: ValueKey(selected),
            padding: EdgeInsets.all(wide ? BPTokens.xxl : BPTokens.lg),
            children: [
              Text(
                selected == "products" ? "Productos" : "Movimientos",
                style: theme.heading,
              ),
              const SizedBox(height: BPTokens.sm),
              Text(
                selected == "products"
                    ? "Consulta de existencias de ejemplo"
                    : "Vista de demostración; sin operaciones reales",
                style: theme.muted,
              ),
              const SizedBox(height: BPTokens.xl),
              if (selected == "products") ...[
                Text("Buscar por nombre o código", style: theme.label),
                const SizedBox(height: BPTokens.sm),
                BPInput(
                  label: "Buscar productos",
                  controller: search,
                  placeholder: "Ej. café o ALI-002",
                  onChanged: (value) =>
                      setState(() => query = value.trim().toLowerCase()),
                ),
                const SizedBox(height: BPTokens.lg),
                Text("${visible.length} productos", style: theme.muted),
                const SizedBox(height: BPTokens.md),
                if (visible.isEmpty)
                  BPCard(
                    child: Column(
                      children: [
                        const Text("No encontramos productos"),
                        const SizedBox(height: BPTokens.md),
                        BPButton(
                          primary: false,
                          label: "Limpiar búsqueda",
                          onPressed: () {
                            search.clear();
                            setState(() => query = "");
                          },
                        ),
                      ],
                    ),
                  )
                else
                  ...visible.map(
                    (p) => Padding(
                      padding: const EdgeInsets.only(bottom: BPTokens.sm),
                      child: BPCard(
                        child: wide
                            ? Row(
                                children: [
                                  Expanded(
                                    flex: 3,
                                    child: Text(p.$1, style: theme.title),
                                  ),
                                  Expanded(flex: 2, child: Text(p.$2)),
                                  Expanded(child: Text("${p.$3} und.")),
                                  Expanded(flex: 2, child: Text(p.$4)),
                                ],
                              )
                            : Column(
                                crossAxisAlignment: CrossAxisAlignment.start,
                                children: [
                                  Text(p.$1, style: theme.title),
                                  const SizedBox(height: BPTokens.xs),
                                  Text(p.$2, style: theme.muted),
                                  const SizedBox(height: BPTokens.sm),
                                  Text("${p.$3} und. · ${p.$4}"),
                                ],
                              ),
                      ),
                    ),
                  ),
              ] else
                const BPCard(
                  child: Text(
                    "Los movimientos estarán disponibles al conectar la API protegida.",
                  ),
                ),
            ],
          );
        },
      ),
    );
  }

  Widget _dashboard(BPVisuals theme, bool wide) {
    final cards = [
      for (final metric in const [
        ("Productos de ejemplo", "3"),
        ("Stock bajo de ejemplo", "1"),
        ("Agotados de ejemplo", "1"),
      ])
        BPCard(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              Text(metric.$1, style: theme.label),
              const SizedBox(height: BPTokens.sm),
              Text(metric.$2, style: theme.heading),
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
