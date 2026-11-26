import "package:flutter/widgets.dart";

import "../bp_table.dart";

/// Synthetic data only. Demonstrates the controlled page contract.
class BPTableExample extends StatefulWidget {
  const BPTableExample({super.key});
  @override
  State<BPTableExample> createState() => _BPTableExampleState();
}

class _BPTableExampleState extends State<BPTableExample> {
  final search = TextEditingController();
  static const pageSize = 3;
  final fixtures = List.generate(
    9,
    (i) => BPTableRow("demo-${i + 1}", ["Producto ${i + 1}", "SKU-${i + 1}"]),
  );
  String? selected, inspected;
  int page = 0;

  @override
  void dispose() {
    search.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final query = search.text.trim().toLowerCase();
    final matches = fixtures
        .where((r) => r.cells.any((c) => c.toLowerCase().contains(query)))
        .toList();
    final pages = (matches.length + pageSize - 1) ~/ pageSize;
    return BPTable(
      columns: const ["Producto", "Código"],
      rows: matches.skip(page * pageSize).take(pageSize).toList(),
      page: page,
      pageCount: pages,
      searchController: search,
      onSearch: (_) => setState(() {
        page = 0;
        selected = null;
      }),
      selectedId: selected,
      onRowDetails: (id) => setState(() => inspected = id),
      toolbar: inspected == null
          ? null
          : Text("Detalle de ejemplo: $inspected"),
      onSelect: (id) => setState(() => selected = id),
      onPage: (index) => setState(() {
        page = index;
        selected = null;
      }),
    );
  }
}
