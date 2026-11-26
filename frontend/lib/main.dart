import "package:flutter/widgets.dart";

void main() => runApp(const BalanzaProApp());

/// Minimal development shell. No API, credentials or business operations.
class BalanzaProApp extends StatelessWidget {
  const BalanzaProApp({super.key});

  @override
  Widget build(BuildContext context) {
    return WidgetsApp(
      title: "BalanzaPro3.0",
      color: const Color(0xFFFFFFFF),
      builder: (context, child) => const _DevelopmentHome(),
    );
  }
}

class _DevelopmentHome extends StatelessWidget {
  const _DevelopmentHome();

  @override
  Widget build(BuildContext context) {
    return const ColoredBox(
      color: Color(0xFFFFFFFF),
      child: SafeArea(
        child: Center(
          child: Padding(
            padding: EdgeInsets.all(24),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                Text(
                  "BalanzaPro3.0",
                  style: TextStyle(
                    color: Color(0xFF17212B),
                    fontSize: 26,
                    fontWeight: FontWeight.w600,
                  ),
                ),
                SizedBox(height: 12),
                Text(
                  "Entorno de desarrollo · Datos ficticios",
                  textAlign: TextAlign.center,
                  style: TextStyle(color: Color(0xFF465563), fontSize: 16),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}
