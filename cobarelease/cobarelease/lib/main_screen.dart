import 'package:flutter/material.dart';
import 'package:secpanel/components/export/export_bottom_sheet.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:secpanel/components/panel/add/add_panel_bottom_sheet.dart';
import 'package:secpanel/components/import/import_bottom_sheet.dart';
import 'package:secpanel/custom_bottom_navbar.dart';
import 'package:secpanel/helpers/db_helper.dart';
import 'package:secpanel/home.dart';
import 'package:secpanel/models/approles.dart';
import 'package:secpanel/models/company.dart';
import 'package:secpanel/profile.dart';
import 'dart:convert';
import 'dart:io';
import 'package:file_picker/file_picker.dart';
import 'package:flutter/foundation.dart';
import 'package:intl/intl.dart';
import 'package:path_provider/path_provider.dart';
import 'package:share_plus/share_plus.dart';
import 'package:secpanel/theme/colors.dart';

class MainScreen extends StatefulWidget {
  const MainScreen({super.key});

  @override
  State<MainScreen> createState() => _MainScreenState();
}

class _MainScreenState extends State<MainScreen> {
  int _selectedIndex = 0;
  Company? _currentCompany;
  List<Company> _k3Vendors = [];
  List<Widget> _pages = [];
  bool _isLoading = true;

  @override
  void initState() {
    super.initState();
    _loadCompanyDataAndInitializePages();
  }

  Future<void> _loadCompanyDataAndInitializePages() async {
    final prefs = await SharedPreferences.getInstance();
    final companyId = prefs.getString('companyId');

    if (companyId == null) {
      if (mounted) {
        Navigator.of(context).pushReplacementNamed('/login');
      }
      return;
    }

    final company = await DatabaseHelper.instance.getCompanyById(companyId);

    if (company == null) {
      if (mounted) {
        Navigator.of(context).pushReplacementNamed('/login');
      }
      return;
    }

    final k3Vendors = await DatabaseHelper.instance.getK3Vendors();

    if (mounted) {
      setState(() {
        _currentCompany = company;
        _k3Vendors = k3Vendors;
        _pages = [
          HomeScreen(
            currentCompany: _currentCompany!,
            onRefresh: _refreshHomeScreen,
          ),
          const ProfileScreen(),
        ];
        _isLoading = false;
      });
    }
  }

  void _showExportBottomSheet() async {
    final result = await showModalBottomSheet<Map<String, dynamic>>(
      context: context,
      isScrollControlled: true,
      backgroundColor: Colors.white,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(16)),
      ),
      builder: (context) {
        return PreviewBottomSheet(currentUser: _currentCompany!);
      },
    );

    if (result != null && mounted) {
      await _processExport(result);
    }
  }

  // --- [PERUBAHAN] Fungsi ini dirombak total untuk menangani format ekspor baru ---
  Future<void> _processExport(Map<String, dynamic> exportData) async {
    final bool exportPanel = exportData['exportPanel'] as bool? ?? false;
    final bool exportUser = exportData['exportUser'] as bool? ?? false;
    final format = exportData['format'] as String;

    if (!exportPanel && !exportUser) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(
            content: Text('Tidak ada data yang dipilih untuk diekspor.'),
            backgroundColor: Colors.orange,
          ),
        );
      }
      return;
    }

    showDialog(
      context: context,
      barrierDismissible: false,
      builder: (BuildContext context) {
        return const Dialog(
          child: Padding(
            padding: EdgeInsets.all(20.0),
            child: Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                CircularProgressIndicator(color: AppColors.schneiderGreen),
                SizedBox(width: 20),
                Text("Mengekspor data..."),
              ],
            ),
          ),
        );
      },
    );

    String? successMessage;
    String? errorMessage;

    try {
      final timestamp = DateFormat('ddMMyy_HHmmss').format(DateTime.now());
      String extension;
      List<int>? fileBytes;
      final Company currentUser = _currentCompany!;

      switch (format) {
        case 'Excel':
          extension = 'xlsx';
          final excel = await DatabaseHelper.instance.generateCustomExportExcel(
            includePanelData: exportPanel,
            includeUserData: exportUser,
            currentUser: currentUser,
          );
          fileBytes = excel.encode();
          break;
        case 'JSON':
          extension = 'json';
          final jsonString = await DatabaseHelper.instance
              .generateCustomExportJson(
                includePanelData: exportPanel,
                includeUserData: exportUser,
                currentUser: currentUser,
              );
          fileBytes = utf8.encode(jsonString);
          break;
        default:
          throw Exception("Format tidak dikenal");
      }

      final fileName = "ExportDataPanel_$timestamp.$extension";
      String? selectedPath;

      if (!kIsWeb &&
          (Platform.isAndroid ||
              Platform.isIOS ||
              Platform.isWindows ||
              Platform.isMacOS)) {
        selectedPath = await FilePicker.platform.getDirectoryPath();
      }

      if (selectedPath != null) {
        final targetDir = Directory(selectedPath);
        final path = "${targetDir.path}/$fileName";

        if (fileBytes != null) {
          final file = File(path)..createSync(recursive: true);
          await file.writeAsBytes(fileBytes);
          successMessage = "File berhasil disimpan: $fileName";

          if (Platform.isIOS || Platform.isMacOS) {
            await Share.shareXFiles([XFile(path)], text: 'File Ekspor Data');
          }
        } else {
          throw Exception("Gagal membuat data file.");
        }
      } else {
        errorMessage = "Ekspor dibatalkan: Tidak ada folder yang dipilih.";
      }
    } catch (e) {
      errorMessage = "Ekspor gagal: $e";
    } finally {
      if (mounted) {
        Navigator.of(context).pop();
      }
    }

    if (mounted) {
      if (successMessage != null) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
            content: Text(successMessage),
            duration: const Duration(seconds: 4),
            backgroundColor: AppColors.schneiderGreen,
          ),
        );
      }
      if (errorMessage != null) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(errorMessage), backgroundColor: Colors.red),
        );
      }
    }
  }
  // --- [AKHIR PERUBAHAN] ---

  void _refreshHomeScreen() {
    setState(() {
      _pages[0] = HomeScreen(
        key: UniqueKey(),
        currentCompany: _currentCompany!,
        onRefresh: _refreshHomeScreen,
      );
    });
  }

  void _onItemTapped(int index) {
    setState(() {
      _selectedIndex = index;
    });
  }

  void _openAddPanelSheet() {
    showModalBottomSheet(
      context: context,
      isScrollControlled: true,
      backgroundColor: Colors.white,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(16)),
      ),
      builder: (_) => AddPanelBottomSheet(
        currentCompany: _currentCompany!,
        k3Vendors: _k3Vendors,
        onPanelAdded: (newPanel) {
          _refreshHomeScreen();
        },
      ),
    );
  }

  void _showImportBottomSheet() {
    showModalBottomSheet(
      context: context,
      isScrollControlled: true,
      backgroundColor: Colors.white,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(16)),
      ),
      builder: (context) {
        return ImportBottomSheet(
          onImportSuccess: () {
            _refreshHomeScreen();
            ScaffoldMessenger.of(context).showSnackBar(
              const SnackBar(
                content: Text('Data berhasil diperbarui!'),
                backgroundColor: Colors.green,
              ),
            );
          },
        );
      },
    );
  }

  @override
  Widget build(BuildContext context) {
    if (_isLoading) {
      return const Scaffold(
        body: Center(
          child: CircularProgressIndicator(color: AppColors.schneiderGreen),
        ),
      );
    }

    final bool canAddPanel =
        _currentCompany?.role == AppRole.admin ||
        _currentCompany?.role == AppRole.k3;
    final bool canImportData =
        _currentCompany?.role == AppRole.admin ||
        _currentCompany?.role == AppRole.k3;
    final bool canExportData = true;

    return Scaffold(
      backgroundColor: AppColors.white,
      body: Stack(
        children: [
          IndexedStack(index: _selectedIndex, children: _pages),
          Positioned(
            bottom: 20,
            right: 16,
            child: PopupMenuButton<String>(
              offset: const Offset(0, -100),
              color: AppColors.white,
              elevation: 2,
              shape: RoundedRectangleBorder(
                borderRadius: BorderRadius.circular(8),
                side: const BorderSide(color: AppColors.grayLight, width: 2),
              ),
              itemBuilder: (BuildContext context) {
                List<PopupMenuEntry<String>> items = [];
                if (canImportData) {
                  items.add(
                    PopupMenuItem<String>(
                      value: 'import',
                      height: 36,
                      child: Row(
                        children: [
                          Image.asset(
                            'assets/images/import-green.png',
                            width: 24,
                            height: 24,
                            color: AppColors.schneiderGreen,
                          ),
                          const SizedBox(width: 12),
                          const Text(
                            'Import',
                            style: TextStyle(
                              color: AppColors.black,
                              fontSize: 12,
                              fontWeight: FontWeight.w400,
                            ),
                          ),
                        ],
                      ),
                    ),
                  );
                }
                if (canExportData) {
                  items.add(
                    PopupMenuItem<String>(
                      value: 'export',
                      height: 36,
                      child: Row(
                        children: [
                          Image.asset(
                            'assets/images/export-green.png',
                            width: 24,
                            height: 24,
                            color: AppColors.schneiderGreen,
                          ),
                          const SizedBox(width: 12),
                          const Text(
                            'Export',
                            style: TextStyle(
                              color: AppColors.black,
                              fontSize: 12,
                              fontWeight: FontWeight.w400,
                            ),
                          ),
                        ],
                      ),
                    ),
                  );
                }
                return items;
              },
              onSelected: (String result) {
                switch (result) {
                  case 'import':
                    _showImportBottomSheet();
                    break;
                  case 'export':
                    _showExportBottomSheet();
                    break;
                }
              },
              child: SizedBox(
                height: 52,
                child: FloatingActionButton.extended(
                  heroTag: 'dataMenuFab',
                  onPressed: null,
                  backgroundColor: AppColors.white,
                  elevation: 0.0,
                  shape: const StadiumBorder(
                    side: BorderSide(color: AppColors.grayLight, width: 2),
                  ),
                  icon: Image.asset(
                    'assets/images/import-export-green.png',
                    width: 24,
                    height: 24,
                    color: AppColors.schneiderGreen,
                  ),
                  label: const Text(
                    'Mass Data',
                    style: TextStyle(
                      color: AppColors.black,
                      fontSize: 12,
                      fontWeight: FontWeight.w400,
                    ),
                  ),
                ),
              ),
            ),
          ),
        ],
      ),
      floatingActionButton: canAddPanel
          ? FloatingActionButton(
              heroTag: 'addPanelFab',
              onPressed: _openAddPanelSheet,
              backgroundColor: AppColors.schneiderGreen,
              foregroundColor: AppColors.white,
              shape: const CircleBorder(),
              elevation: 0.0,
              child: const Icon(Icons.add),
            )
          : null,
      floatingActionButtonLocation: FloatingActionButtonLocation.centerDocked,
      bottomNavigationBar: Container(
        height: 70,
        decoration: BoxDecoration(
          color: AppColors.white,
          boxShadow: [
            BoxShadow(
              color: Colors.black.withOpacity(0.05),
              spreadRadius: 1,
              blurRadius: 10,
              offset: const Offset(0, -3),
            ),
          ],
        ),
        child: BottomAppBar(
          color: Colors.transparent,
          elevation: 0,
          shape: const CircularNotchedRectangle(),
          notchMargin: 8.0,
          child: CustomBottomNavBar(
            selectedIndex: _selectedIndex,
            onItemTapped: _onItemTapped,
          ),
        ),
      ),
    );
  }
}
