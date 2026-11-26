import "package:balanzapro_alpha/main.dart";
import "package:flutter_test/flutter_test.dart";

void main() {
  testWidgets("scaffold explicitly identifies synthetic mode", (tester) async {
    await tester.pumpWidget(const BalanzaProApp());
    expect(find.text("BalanzaPro3.0"), findsOneWidget);
    expect(
      find.text("Entorno de desarrollo · Datos ficticios"),
      findsOneWidget,
    );
  });
}
