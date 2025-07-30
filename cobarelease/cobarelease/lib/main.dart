import 'package:flutter/material.dart';
import 'package:secpanel/login.dart';
import 'package:secpanel/login_change_password.dart';
import 'package:secpanel/main_screen.dart';

void main() {
  runApp(const MyApp());
}

class MyApp extends StatelessWidget {
  const MyApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      theme: ThemeData(
        useMaterial3: true,
        fontFamily: 'Lexend',
        textTheme: const TextTheme(
          bodyMedium: TextStyle(fontWeight: FontWeight.w400, fontSize: 14),
          bodyLarge: TextStyle(fontWeight: FontWeight.w500, fontSize: 16),
          titleLarge: TextStyle(fontWeight: FontWeight.w700, fontSize: 20),
        ),
      ),
      title: 'Schneider Indonesia',
      debugShowCheckedModeBanner: false,

      home: const LoginPage(),

      routes: {
        '/login': (context) => const LoginPage(),
        '/home': (context) => const MainScreen(),
        '/login-change-password': (context) => const LoginChangePasswordPage(),
      },
    );
  }
}
