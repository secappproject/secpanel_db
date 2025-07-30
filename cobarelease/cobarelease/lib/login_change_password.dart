import 'package:flutter/material.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:secpanel/helpers/db_helper.dart';
import 'package:secpanel/models/company.dart';
import 'package:secpanel/theme/colors.dart';

class LoginChangePasswordPage extends StatefulWidget {
  const LoginChangePasswordPage({super.key});

  @override
  State<LoginChangePasswordPage> createState() =>
      _LoginChangePasswordPageState();
}

class _LoginChangePasswordPageState extends State<LoginChangePasswordPage> {
  final _usernameController = TextEditingController();
  final _passwordController = TextEditingController();
  final _newPasswordController = TextEditingController();

  bool _isLoading = false;
  bool _isPasswordVisible = false;
  bool _isNewPasswordVisible = false;

  @override
  void dispose() {
    _usernameController.dispose();
    _passwordController.dispose();
    _newPasswordController.dispose();
    super.dispose();
  }

  Future<void> _loginAndChangePassword() async {
    if (_isLoading) return;

    setState(() {
      _isLoading = true;
    });

    try {
      await Future.delayed(const Duration(seconds: 1));

      final username = _usernameController.text.trim();
      final password = _passwordController.text.trim();
      final newPassword = _newPasswordController.text.trim();

      if (username.isEmpty || password.isEmpty || newPassword.isEmpty) {
        _showErrorSnackBar('Semua field tidak boleh kosong.');
        setState(() => _isLoading = false);
        return;
      }

      if (newPassword == password) {
        _showErrorSnackBar(
          'Password baru tidak boleh sama dengan password saat ini.',
        );
        setState(() => _isLoading = false);
        return;
      }

      final Company? company = await DatabaseHelper.instance.login(
        username,
        password,
      );

      if (mounted) {
        if (company != null) {
          final updated = await DatabaseHelper.instance.updatePassword(
            username,
            newPassword,
          );

          if (updated) {
            _showSuccessSnackBar('Password berhasil diubah!');
            final prefs = await SharedPreferences.getInstance();
            await prefs.setString('loggedInUsername', username);
            await prefs.setString('companyId', company.id);
            await prefs.setString('companyRole', company.role.name);

            Navigator.pushReplacementNamed(context, '/home');
          } else {
            _showErrorSnackBar('Gagal mengubah password.');
          }
        } else {
          _showErrorSnackBar('Username atau password saat ini salah.');
        }
        setState(() {
          _isLoading = false;
        });
      }
    } catch (e) {
      if (mounted) {
        _showErrorSnackBar('Terjadi kesalahan: $e');
        setState(() => _isLoading = false);
      }
    }
  }

  void _showErrorSnackBar(String message) {
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(
        content: Text(message),
        backgroundColor: Colors.redAccent,
        behavior: SnackBarBehavior.floating,
      ),
    );
  }

  void _showSuccessSnackBar(String message) {
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(
        content: Text(message),
        backgroundColor: Colors.green,
        behavior: SnackBarBehavior.floating,
      ),
    );
  }

  InputDecoration _buildInputDecoration(
    String labelText, {
    Widget? suffixIcon,
  }) {
    return InputDecoration(
      labelText: labelText,
      labelStyle: const TextStyle(
        fontFamily: 'Lexend',
        fontWeight: FontWeight.w300,
        fontSize: 12,
        color: Colors.black87,
      ),
      filled: true,
      fillColor: AppColors.white,
      contentPadding: const EdgeInsets.symmetric(vertical: 16, horizontal: 16),
      border: OutlineInputBorder(
        borderRadius: BorderRadius.circular(8),
        borderSide: BorderSide.none,
      ),
      focusedBorder: OutlineInputBorder(
        borderRadius: BorderRadius.circular(8),
        borderSide: const BorderSide(color: AppColors.schneiderGreen, width: 2),
      ),
      enabledBorder: OutlineInputBorder(
        borderRadius: BorderRadius.circular(8),
        borderSide: const BorderSide(color: AppColors.grayNeutral, width: 1),
      ),
      suffixIcon: suffixIcon,
    );
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: Colors.white,
      body: SafeArea(
        child: Column(
          mainAxisAlignment: MainAxisAlignment.spaceBetween,
          children: [
            // Bagian atas (Logo dan Form)
            SingleChildScrollView(
              padding: const EdgeInsets.all(16.0),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  const SizedBox(height: 50),
                  // Logo dan Judul
                  Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Image.asset('assets/images/logo.png', height: 44),
                      const SizedBox(height: 24),
                      const Text(
                        'Ubah Password',
                        style: TextStyle(
                          fontFamily: 'Lexend',
                          fontWeight: FontWeight.w400,
                          fontSize: 32,
                          color: AppColors.black,
                        ),
                      ),
                    ],
                  ),
                  const SizedBox(height: 40),
                  // Username TextField
                  TextField(
                    controller: _usernameController,
                    cursorColor: AppColors.schneiderGreen,
                    style: const TextStyle(
                      fontFamily: 'Lexend',
                      fontSize: 14,
                      fontWeight: FontWeight.w300,
                    ),
                    decoration: _buildInputDecoration('Username'),
                  ),
                  const SizedBox(height: 16),
                  // Password Saat Ini
                  TextField(
                    controller: _passwordController,
                    cursorColor: AppColors.schneiderGreen,
                    obscureText: !_isPasswordVisible,
                    style: const TextStyle(
                      fontFamily: 'Lexend',
                      fontSize: 14,
                      fontWeight: FontWeight.w300,
                    ),
                    decoration: _buildInputDecoration(
                      'Password Saat Ini',
                      suffixIcon: IconButton(
                        icon: Icon(
                          _isPasswordVisible
                              ? Icons.visibility_off
                              : Icons.visibility,
                          color: AppColors.gray,
                        ),
                        onPressed: () {
                          setState(() {
                            _isPasswordVisible = !_isPasswordVisible;
                          });
                        },
                      ),
                    ),
                  ),
                  const SizedBox(height: 16),
                  // Password Baru
                  TextField(
                    controller: _newPasswordController,
                    cursorColor: AppColors.schneiderGreen,
                    obscureText: !_isNewPasswordVisible,
                    style: const TextStyle(
                      fontFamily: 'Lexend',
                      fontSize: 14,
                      fontWeight: FontWeight.w300,
                    ),
                    decoration: _buildInputDecoration(
                      'Password Baru',
                      suffixIcon: IconButton(
                        icon: Icon(
                          _isNewPasswordVisible
                              ? Icons.visibility_off
                              : Icons.visibility,
                          color: AppColors.gray,
                        ),
                        onPressed: () {
                          setState(() {
                            _isNewPasswordVisible = !_isNewPasswordVisible;
                          });
                        },
                      ),
                    ),
                  ),
                ],
              ),
            ),
            // Bagian bawah (Tombol) - Dibuat sama dengan LoginPage
            Padding(
              padding: const EdgeInsets.all(16.0),
              child: Column(
                children: [
                  Container(height: 1, color: AppColors.grayNeutral),
                  const SizedBox(height: 16),
                  Row(
                    children: [
                      Expanded(
                        child: OutlinedButton(
                          style: OutlinedButton.styleFrom(
                            minimumSize: const Size(0, 52),
                            side: const BorderSide(
                              color: AppColors.schneiderGreen,
                            ),
                            shape: RoundedRectangleBorder(
                              borderRadius: BorderRadius.circular(6),
                            ),
                          ),
                          onPressed: _isLoading
                              ? null
                              : () {
                                  Navigator.of(context).pop();
                                },
                          child: const Text(
                            'Batal',
                            textAlign: TextAlign.center,
                            style: TextStyle(
                              fontFamily: 'Lexend',
                              fontWeight: FontWeight.w500,
                              fontSize: 12,
                              color: AppColors.schneiderGreen,
                            ),
                          ),
                        ),
                      ),
                      const SizedBox(width: 16),
                      Expanded(
                        child: ElevatedButton(
                          style: ElevatedButton.styleFrom(
                            minimumSize: const Size(double.infinity, 52),
                            shadowColor: Colors.transparent,
                            backgroundColor: AppColors.schneiderGreen,
                            foregroundColor: Colors.white,
                            disabledBackgroundColor: AppColors.schneiderGreen
                                .withOpacity(0.7),
                            shape: RoundedRectangleBorder(
                              borderRadius: BorderRadius.circular(6),
                            ),
                          ),
                          onPressed: _isLoading
                              ? null
                              : _loginAndChangePassword,
                          child: _isLoading
                              ? const SizedBox(
                                  height: 24,
                                  width: 24,
                                  child: CircularProgressIndicator(
                                    color: Colors.white,
                                    strokeWidth: 3,
                                  ),
                                )
                              : const Text(
                                  'Simpan & Masuk',
                                  style: TextStyle(
                                    fontFamily: 'Lexend',
                                    fontWeight: FontWeight.w500,
                                    fontSize: 12,
                                  ),
                                ),
                        ),
                      ),
                    ],
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}
