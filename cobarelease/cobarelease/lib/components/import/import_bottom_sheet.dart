import 'dart:convert';
import 'dart:io';
import 'package:excel/excel.dart';
import 'package:file_picker/file_picker.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:intl/intl.dart';
import 'package:secpanel/components/import/import_review_screen.dart';
import 'package:secpanel/helpers/db_helper.dart';
import 'package:secpanel/theme/colors.dart';

class ImportBottomSheet extends StatefulWidget {
  final VoidCallback onImportSuccess;

  const ImportBottomSheet({super.key, required this.onImportSuccess});

  @override
  State<ImportBottomSheet> createState() => _ImportBottomSheetState();
}

class _ImportBottomSheetState extends State<ImportBottomSheet> {
  bool _isProcessing = false;
  String _statusText = "Ketuk untuk memilih file";
  String _selectedTemplateType = 'panels_and_relations';
  String _selectedTemplateFormat = 'xlsx';
  bool _isDownloading = false;

  Future<void> _downloadTemplate() async {
    if (_isDownloading) return;
    setState(() => _isDownloading = true);

    try {
      final templateFile = await DatabaseHelper.instance.generateImportTemplate(
        dataType: _selectedTemplateType,
        format: _selectedTemplateFormat,
      );
      final String? selectedDirectory = await FilePicker.platform
          .getDirectoryPath(
            dialogTitle: 'Pilih folder untuk menyimpan template',
          );

      if (selectedDirectory != null) {
        final timestamp = DateFormat('yyyyMMdd_HHmmss').format(DateTime.now());
        final fileName =
            'template_import_${_selectedTemplateType}_$timestamp.${templateFile.extension}';
        final filePath = '$selectedDirectory/$fileName';
        final file = File(filePath);
        await file.writeAsBytes(templateFile.bytes);

        if (mounted) {
          ScaffoldMessenger.of(context).showSnackBar(
            const SnackBar(
              content: Text('Template berhasil disimpan.'),
              backgroundColor: AppColors.schneiderGreen,
            ),
          );
        }
      }
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
            content: Text("Gagal mengunduh template: $e"),
            backgroundColor: AppColors.red,
          ),
        );
      }
    } finally {
      if (mounted) setState(() => _isDownloading = false);
    }
  }

  Future<void> _pickAndProcessFile() async {
    if (_isProcessing) return;
    setState(() {
      _isProcessing = true;
      _statusText = "Membuka direktori...";
    });
    try {
      final result = await FilePicker.platform.pickFiles(
        type: FileType.custom,
        allowedExtensions: ['json', 'xlsx'],
      );
      if (result != null) {
        final pickedFile = result.files.single;
        setState(() => _statusText = "Memproses: ${pickedFile.name}");
        await Future.delayed(const Duration(milliseconds: 200));
        final bytes = kIsWeb
            ? pickedFile.bytes!
            : await File(pickedFile.path!).readAsBytes();
        final Map<String, List<Map<String, dynamic>>> data =
            pickedFile.extension == 'json'
            ? _parseJson(bytes)
            : _parseExcel(bytes);

        if (mounted) {
          Navigator.pop(context);
          final importFinished = await Navigator.push<bool>(
            context,
            MaterialPageRoute(
              builder: (context) => ImportReviewScreen(initialData: data),
            ),
          );
          if (importFinished == true) {
            widget.onImportSuccess();
          }
        }
      } else {
        setState(() {
          _isProcessing = false;
          _statusText = "Ketuk untuk memilih file";
        });
      }
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
            content: Text("Gagal memproses file: $e"),
            backgroundColor: Colors.red,
          ),
        );
        setState(() {
          _isProcessing = false;
          _statusText = "Ketuk untuk memilih file";
        });
      }
    }
  }

  Map<String, List<Map<String, dynamic>>> _parseJson(Uint8List bytes) {
    final content = utf8.decode(bytes);
    final jsonData = json.decode(content) as Map<String, dynamic>;
    final Map<String, List<Map<String, dynamic>>> result = {};
    jsonData.forEach((key, value) {
      if (value is List) {
        result[key.toLowerCase()] = value.cast<Map<String, dynamic>>();
      }
    });
    return result;
  }

  Map<String, List<Map<String, dynamic>>> _parseExcel(Uint8List bytes) {
    final excel = Excel.decodeBytes(bytes);
    final Map<String, List<Map<String, dynamic>>> result = {};
    for (var tableName in excel.tables.keys) {
      final lowerCaseTableName = tableName.toLowerCase().replaceAll(' ', '_');

      final sheet = excel.tables[tableName]!;
      if (sheet.maxRows <= 1) {
        result[lowerCaseTableName] = [];
        continue;
      }
      final List<String> header = sheet.rows.first
          .map((cell) => cell?.value?.toString().trim().toLowerCase() ?? '')
          .toList();
      final List<Map<String, dynamic>> sheetRows = [];
      for (int i = 1; i < sheet.maxRows; i++) {
        final row = sheet.rows[i];
        final rowData = <String, dynamic>{};
        bool isRowCompletelyEmpty = true;
        for (int j = 0; j < header.length; j++) {
          final key = header[j];
          if (key.isNotEmpty) {
            final dynamic rawValue = (j < row.length) ? row[j]?.value : null;
            final String cellValueAsString = rawValue?.toString() ?? '';
            rowData[key] = cellValueAsString;
            if (cellValueAsString.trim().isNotEmpty) {
              isRowCompletelyEmpty = false;
            }
          }
        }
        if (!isRowCompletelyEmpty) sheetRows.add(rowData);
      }
      result[lowerCaseTableName] = sheetRows;
    }
    return result;
  }

  @override
  Widget build(BuildContext context) {
    return SingleChildScrollView(
      child: Padding(
        padding: EdgeInsets.fromLTRB(
          20,
          16,
          20,
          MediaQuery.of(context).viewInsets.bottom + 24,
        ),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Center(
              child: Container(
                height: 5,
                width: 40,
                decoration: BoxDecoration(
                  color: AppColors.grayLight,
                  borderRadius: BorderRadius.circular(100),
                ),
              ),
            ),
            const SizedBox(height: 24),
            _buildTemplateDownloaderCard(),
            const Divider(height: 40, color: AppColors.grayLight),
            const Text(
              "Impor Data",
              style: TextStyle(fontSize: 20, fontWeight: FontWeight.w400),
            ),
            const SizedBox(height: 8),
            const Text(
              "Pilih file .xlsx atau .json untuk memulai proses impor.",
              style: TextStyle(fontSize: 12, color: AppColors.gray),
            ),
            const SizedBox(height: 24),
            InkWell(
              onTap: _pickAndProcessFile,
              borderRadius: BorderRadius.circular(8),
              child: Container(
                width: double.infinity,
                padding: const EdgeInsets.symmetric(vertical: 24),
                decoration: BoxDecoration(
                  borderRadius: BorderRadius.circular(8),
                  border: BoxBorder.all(color: AppColors.grayLight),
                ),
                child: _isProcessing
                    ? Column(
                        mainAxisAlignment: MainAxisAlignment.center,
                        children: [
                          const SizedBox(
                            width: 24,
                            height: 24,
                            child: CircularProgressIndicator(
                              strokeWidth: 3,
                              color: AppColors.schneiderGreen,
                            ),
                          ),
                          const SizedBox(height: 16),
                          Text(
                            _statusText,
                            textAlign: TextAlign.center,
                            style: const TextStyle(color: AppColors.gray),
                          ),
                        ],
                      )
                    : Column(
                        mainAxisAlignment: MainAxisAlignment.center,
                        children: [
                          Image.asset(
                            'assets/images/import-green.png',
                            height: 24,
                          ),
                          const SizedBox(height: 8),
                          Text(
                            _statusText,
                            style: const TextStyle(color: AppColors.gray),
                          ),
                        ],
                      ),
              ),
            ),
            const SizedBox(height: 24),
            SizedBox(
              width: double.infinity,
              child: OutlinedButton(
                onPressed: () => Navigator.of(context).pop(),
                style: OutlinedButton.styleFrom(
                  padding: const EdgeInsets.symmetric(vertical: 16),
                  side: const BorderSide(color: AppColors.grayLight),
                ),
                child: const Text(
                  "Batal",
                  style: TextStyle(color: AppColors.black),
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildTemplateDownloaderCard() {
    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        borderRadius: BorderRadius.circular(8),
        border: BoxBorder.all(color: AppColors.grayLight),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Image.asset('assets/images/export-green.png', height: 24),
              const SizedBox(width: 8),
              const Text(
                "Unduh Template Impor",
                style: TextStyle(fontSize: 16, fontWeight: FontWeight.w500),
              ),
            ],
          ),
          const Divider(height: 24, color: AppColors.grayLight),
          _buildSectionTitle("1. Pilih Tipe Data"),
          Row(
            children: [
              _buildOptionButton(
                'Panel & Relasi',
                'panels_and_relations',
                _selectedTemplateType,
                (val) => setState(() => _selectedTemplateType = val),
              ),
              _buildOptionButton(
                'Company & Akun',
                'companies_and_accounts',
                _selectedTemplateType,
                (val) => setState(() => _selectedTemplateType = val),
              ),
            ],
          ),
          const SizedBox(height: 16),
          _buildSectionTitle("2. Pilih Format File"),
          Row(
            children: [
              _buildOptionButton(
                'Excel (.xlsx)',
                'xlsx',
                _selectedTemplateFormat,
                (val) => setState(() => _selectedTemplateFormat = val),
              ),
              _buildOptionButton(
                'JSON (.json)',
                'json',
                _selectedTemplateFormat,
                (val) => setState(() => _selectedTemplateFormat = val),
              ),
            ],
          ),
          const SizedBox(height: 24),
          SizedBox(
            width: double.infinity,
            child: ElevatedButton(
              onPressed: _downloadTemplate,
              style: ElevatedButton.styleFrom(
                padding: const EdgeInsets.symmetric(vertical: 16),
                backgroundColor: AppColors.schneiderGreen,
                foregroundColor: Colors.white,
                elevation: 0,
              ),
              child: _isDownloading
                  ? const SizedBox(
                      height: 16,
                      width: 16,
                      child: CircularProgressIndicator(
                        color: Colors.white,
                        strokeWidth: 2,
                      ),
                    )
                  : const Text(
                      'Unduh Template',
                      style: TextStyle(fontSize: 12),
                    ),
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildSectionTitle(String title) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 12.0),
      child: Text(
        title,
        style: const TextStyle(
          fontSize: 12,
          color: AppColors.gray,
          fontWeight: FontWeight.w400,
        ),
      ),
    );
  }

  Widget _buildOptionButton(
    String label,
    String value,
    String groupValue,
    Function(String) onSelected,
  ) {
    final bool isSelected = value == groupValue;
    return Expanded(
      child: GestureDetector(
        onTap: () => onSelected(value),
        child: Container(
          padding: const EdgeInsets.symmetric(vertical: 12),
          margin: const EdgeInsets.only(right: 8),
          decoration: BoxDecoration(
            color: isSelected
                ? AppColors.schneiderGreen.withOpacity(0.08)
                : AppColors.white,
            border: BoxBorder.all(
              color: isSelected
                  ? AppColors.schneiderGreen
                  : AppColors.grayLight,
            ),
            borderRadius: BorderRadius.circular(8),
          ),
          child: Center(
            child: Text(
              label,
              style: TextStyle(
                color: isSelected ? AppColors.schneiderGreen : AppColors.black,
                fontSize: 12,
              ),
            ),
          ),
        ),
      ),
    );
  }
}
