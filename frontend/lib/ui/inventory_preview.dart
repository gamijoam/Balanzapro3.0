import "package:flutter/widgets.dart";
import "package:shadcn_ui/shadcn_ui.dart";

/// Experimental UI adapter, not the final design system.
class PreviewApp extends StatelessWidget {
  const PreviewApp({super.key});
  @override
  Widget build(BuildContext context) => ShadApp(
    title: "BalanzaPro3.0",
    theme: ShadThemeData(
      brightness: Brightness.light,
      colorScheme: const ShadZincColorScheme.light(),
    ),
    darkTheme: ShadThemeData(
      brightness: Brightness.dark,
      colorScheme: const ShadZincColorScheme.dark(),
    ),
    home: const _InventoryPreview(),
  );
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
  int selected = 0;
  @override
  void dispose() {
    search.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final theme = ShadTheme.of(context);
    final colors = theme.colorScheme;
    final visible = products
        .where(
          (p) =>
              p.$1.toLowerCase().contains(query) ||
              p.$2.toLowerCase().contains(query),
        )
        .toList();
    return ColoredBox(
      color: colors.background,
      child: SafeArea(
        child: LayoutBuilder(
          builder: (context, constraints) {
            final wide = constraints.maxWidth >= 1000;
            return Column(
              children: [
                Container(
                  padding: const EdgeInsets.symmetric(
                    horizontal: 16,
                    vertical: 12,
                  ),
                  decoration: BoxDecoration(
                    border: Border(bottom: BorderSide(color: colors.border)),
                  ),
                  child: Row(
                    children: [
                      Expanded(
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            Text("BalanzaPro3.0", style: theme.textTheme.large),
                            Text(
                              "Entorno de desarrollo · Datos ficticios",
                              style: theme.textTheme.muted,
                            ),
                          ],
                        ),
                      ),
                    ],
                  ),
                ),
                if (!wide) _navigation(false),
                Expanded(
                  child: Row(
                    children: [
                      if (wide) SizedBox(width: 224, child: _navigation(true)),
                      Expanded(
                        child: ListView(
                          padding: EdgeInsets.all(wide ? 32 : 16),
                          children: [
                            Text(
                              selected == 0 ? "Productos" : "Movimientos",
                              style: theme.textTheme.h3,
                            ),
                            const SizedBox(height: 8),
                            Text(
                              selected == 0
                                  ? "Consulta de existencias de ejemplo"
                                  : "Vista de demostración; sin operaciones reales",
                              style: theme.textTheme.muted,
                            ),
                            const SizedBox(height: 24),
                            if (selected == 0) ...[
                              Text(
                                "Buscar por nombre o código",
                                style: theme.textTheme.small,
                              ),
                              const SizedBox(height: 8),
                              Semantics(
                                label: "Buscar productos",
                                child: ShadInput(
                                  controller: search,
                                  placeholder: const Text("Ej. café o ALI-002"),
                                  onChanged: (value) => setState(
                                    () => query = value.trim().toLowerCase(),
                                  ),
                                ),
                              ),
                              const SizedBox(height: 16),
                              Text(
                                "${visible.length} productos",
                                style: theme.textTheme.muted,
                              ),
                              const SizedBox(height: 12),
                              if (visible.isEmpty)
                                ShadCard(
                                  child: Column(
                                    children: [
                                      const Text("No encontramos productos"),
                                      const SizedBox(height: 12),
                                      ShadButton.outline(
                                        onPressed: () {
                                          search.clear();
                                          setState(() => query = "");
                                        },
                                        child: const Text("Limpiar búsqueda"),
                                      ),
                                    ],
                                  ),
                                )
                              else
                                ...visible.map(
                                  (p) => Padding(
                                    padding: const EdgeInsets.only(bottom: 8),
                                    child: ShadCard(
                                      child: wide
                                          ? Row(
                                              children: [
                                                Expanded(
                                                  flex: 3,
                                                  child: Text(
                                                    p.$1,
                                                    style:
                                                        theme.textTheme.large,
                                                  ),
                                                ),
                                                Expanded(
                                                  flex: 2,
                                                  child: Text(p.$2),
                                                ),
                                                Expanded(
                                                  child: Text("${p.$3} und."),
                                                ),
                                                Expanded(
                                                  flex: 2,
                                                  child: Text(p.$4),
                                                ),
                                              ],
                                            )
                                          : Column(
                                              crossAxisAlignment:
                                                  CrossAxisAlignment.start,
                                              children: [
                                                Text(
                                                  p.$1,
                                                  style: theme.textTheme.large,
                                                ),
                                                const SizedBox(height: 4),
                                                Text(
                                                  p.$2,
                                                  style: theme.textTheme.muted,
                                                ),
                                                const SizedBox(height: 8),
                                                Text("${p.$3} und. · ${p.$4}"),
                                              ],
                                            ),
                                    ),
                                  ),
                                ),
                            ] else
                              const ShadCard(
                                child: Text(
                                  "Los movimientos estarán disponibles al conectar la API protegida.",
                                ),
                              ),
                          ],
                        ),
                      ),
                    ],
                  ),
                ),
              ],
            );
          },
        ),
      ),
    );
  }

  Widget _navigation(bool wide) {
    final items = [_navButton("Productos", 0), _navButton("Movimientos", 1)];
    return Padding(
      key: Key(wide ? "desktop-navigation" : "compact-navigation"),
      padding: const EdgeInsets.all(12),
      child: wide
          ? Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: items,
            )
          : Wrap(spacing: 8, runSpacing: 8, children: items),
    );
  }

  Widget _navButton(String label, int index) {
    void activate() => setState(() => selected = index);
    return Padding(
      padding: const EdgeInsets.only(bottom: 4),
      child: Semantics(
        selected: selected == index,
        child: SizedBox(
          height: 48,
          child: selected == index
              ? ShadButton(onPressed: activate, child: Text(label))
              : ShadButton.outline(onPressed: activate, child: Text(label)),
        ),
      ),
    );
  }
}
