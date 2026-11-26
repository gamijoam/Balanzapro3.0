import "package:flutter/widgets.dart";

import "design_system/design_system.dart";
import "design_system/bp_badge.dart";
import "design_system/bp_page_layout.dart";

void main() => runApp(const BPApp(home: BPIdentityExample()));

class BPIdentityExample extends StatefulWidget {
  const BPIdentityExample({super.key});
  @override
  State<BPIdentityExample> createState() => _BPIdentityExampleState();
}

class _BPIdentityExampleState extends State<BPIdentityExample> {
  String selected = "products";
  @override
  Widget build(BuildContext context) {
    final theme = BPVisuals.of(context);
    return BPPageLayout(
      title: "BalanzaPro3.0",
      contextLabel: "Catálogo de identidad · Datos ficticios",
      items: const [
        BPNavItem("home", "Inicio", icon: BPNavIcon.home),
        BPNavItem("products", "Productos", icon: BPNavIcon.products),
        BPNavItem("movements", "Movimientos", icon: BPNavIcon.movements),
      ],
      selectedId: selected,
      onNavigate: (id) => setState(() => selected = id),
      child: ListView(
        padding: const EdgeInsets.all(BPTokens.lg),
        children: [
          Text("Jerarquía visual", style: theme.heading),
          const SizedBox(height: BPTokens.sm),
          Text("Título de sección", style: theme.title),
          const SizedBox(height: BPTokens.sm),
          Text("Etiqueta de campo", style: theme.label),
          Text("Información complementaria", style: theme.muted),
          const SizedBox(height: BPTokens.lg),
          BPCard(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                Text("Acciones y estados", style: theme.title),
                const SizedBox(height: BPTokens.md),
                BPButton(
                  label: "Acción principal de ejemplo",
                  onPressed: () => setState(() => selected = "products"),
                ),
                const SizedBox(height: BPTokens.sm),
                BPButton(
                  label: "Acción secundaria de ejemplo",
                  primary: false,
                  onPressed: () => setState(() => selected = "home"),
                ),
                const SizedBox(height: BPTokens.md),
                const Wrap(
                  spacing: BPTokens.sm,
                  runSpacing: BPTokens.sm,
                  children: [
                    BPBadge(label: "Disponible", tone: BPTone.success),
                    BPBadge(label: "Stock bajo", tone: BPTone.warning),
                    BPBadge(label: "Agotado", tone: BPTone.danger),
                  ],
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}
