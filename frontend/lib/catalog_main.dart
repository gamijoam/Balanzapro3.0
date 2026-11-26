import "package:flutter/widgets.dart";

import "design_system/catalog/catalog.dart";
import "design_system/design_system.dart";

void main() => runApp(const BPCatalogApp());

/// Development-only entrypoint; not registered in the inventory app.
class BPCatalogApp extends StatelessWidget {
  const BPCatalogApp({super.key});
  @override
  Widget build(BuildContext context) => const BPApp(home: BPCatalog());
}
