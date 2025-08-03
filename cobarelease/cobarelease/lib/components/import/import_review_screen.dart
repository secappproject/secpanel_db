import 'dart:convert'; // <-- Tambahkan import ini
import 'package:flutter/material.dart';
import 'package:intl/intl.dart';
import 'package:secpanel/components/import/confirm_import_bottom_sheet.dart';
import 'package:secpanel/components/import/import_progress_dialog.dart';
import 'package:secpanel/helpers/db_helper.dart';
import 'package:secpanel/models/approles.dart';
import 'package:secpanel/models/company.dart';
import 'package:secpanel/theme/colors.dart';
import 'package:shared_preferences/shared_preferences.dart';

class ImportReviewScreen extends StatefulWidget {
  final Map<String, List<Map<String, dynamic>>> initialData;
  final bool isCustomTemplate;

  const ImportReviewScreen({
    super.key,
    required this.initialData,
    this.isCustomTemplate = false,
  });

  @override
  State<ImportReviewScreen> createState() => _ImportReviewScreenState();
}

class _ImportReviewScreenState extends State<ImportReviewScreen> {
  late Map<String, List<Map<String, dynamic>>> _editableData;
  late Map<String, Set<int>> _duplicateRows;
  late Map<String, Map<int, Set<String>>> _brokenRelationCells;
  bool _isLoading = true;

  late Map<String, Set<String>> _existingPrimaryKeys;
  List<Company> _allCompanies = [];

  static const Map<String, List<String>> _templateColumns = {
    'panel': [
      'PP Panel',
      'Panel No',
      'WBS',
      'PROJECT',
      'Plan Start',
      'Actual Delivery ke SEC',
      'Panel',
      'Busbar',
    ],
    'user': ['Username', 'Password', 'Company', 'Company Role'],
  };

  final ValueNotifier<double> _progressNotifier = ValueNotifier(0.0);
  final ValueNotifier<String> _statusNotifier = ValueNotifier('');

  @override
  void initState() {
    super.initState();
    _editableData = widget.initialData.map((key, value) {
      return MapEntry(
        key,
        value.map((item) => Map<String, dynamic>.from(item)).toList(),
      );
    });
    _duplicateRows = {};
    _brokenRelationCells = {};
    _initializeAndValidateData();
  }

  Future<void> _initializeAndValidateData() async {
    if (mounted) setState(() => _isLoading = true);

    await _fetchAllCompanies();
    await _fetchExistingPrimaryKeys();
    _revalidateOnDataChange();

    if (mounted) setState(() => _isLoading = false);
  }

  Future<void> _fetchAllCompanies() async {
    _allCompanies = await DatabaseHelper.instance.getAllCompanies();
  }

  Future<void> _fetchExistingPrimaryKeys() async {
    final dbHelper = DatabaseHelper.instance;
    _existingPrimaryKeys = {
      'companies': (await dbHelper.getAllCompanies()).map((c) => c.id).toSet(),
      'company_accounts': (await dbHelper.getAllCompanyAccounts())
          .map((a) => a.username)
          .toSet(),
      'panels': (await dbHelper.getAllPanels()).map((p) => p.noPp).toSet(),
      'busbars': (await dbHelper.getAllBusbars())
          .map((b) => "${b.panelNoPp}_${b.vendor}")
          .toSet(),
      'components': (await dbHelper.getAllComponents())
          .map((c) => "${c.panelNoPp}_${c.vendor}")
          .toSet(),
      'palet': (await dbHelper.getAllPalet())
          .map((c) => "${c.panelNoPp}_${c.vendor}")
          .toSet(),
      'corepart': (await dbHelper.getAllCorepart())
          .map((c) => "${c.panelNoPp}_${c.vendor}")
          .toSet(),
    };
  }

  void _validateDuplicates() {
    _duplicateRows = {};
    const Map<String, String> primaryKeyMapping = {
      'companies': 'id',
      'company_accounts': 'username',
      'panels': 'no_pp',
    };
    for (var entry in primaryKeyMapping.entries) {
      final tableName = entry.key;
      final pkColumn = entry.value;
      if (_editableData.containsKey(tableName) &&
          _editableData[tableName]!.isNotEmpty &&
          (_editableData[tableName]!.first.containsKey(pkColumn))) {
        _duplicateRows.putIfAbsent(tableName, () => <int>{});
        final rows = _editableData[tableName]!;
        final pksInDb = _existingPrimaryKeys[tableName] ?? {};
        final pksInFile = <String>{};
        for (int i = 0; i < rows.length; i++) {
          final pkValue = rows[i][pkColumn]?.toString();
          if (pkValue != null && pkValue.isNotEmpty) {
            if (pksInDb.contains(pkValue) || !pksInFile.add(pkValue)) {
              _duplicateRows[tableName]!.add(i);
            }
          }
        }
      }
    }
    final List<String> compositeKeyTables = [
      'busbars',
      'components',
      'palet',
      'corepart',
    ];
    for (final tableName in compositeKeyTables) {
      if (!_editableData.containsKey(tableName) ||
          _editableData[tableName]!.isEmpty)
        continue;
      _duplicateRows.putIfAbsent(tableName, () => <int>{});
      final rows = _editableData[tableName]!;
      final existingCompositeKeys =
          _existingPrimaryKeys[tableName] ?? <String>{};
      final seenKeysInFile = <String>{};
      for (int i = 0; i < rows.length; i++) {
        final row = rows[i];
        final panelNoPp = row['panel_no_pp']?.toString() ?? '';
        final vendor = row['vendor']?.toString() ?? '';
        if (panelNoPp.isNotEmpty && vendor.isNotEmpty) {
          final compositeKey = "${panelNoPp}_${vendor}";
          if (existingCompositeKeys.contains(compositeKey) ||
              !seenKeysInFile.add(compositeKey)) {
            _duplicateRows[tableName]!.add(i);
          }
        }
      }
    }
  }

  void _validateBrokenRelations() {
    _brokenRelationCells = {};
    final allCompanyIDsInFile =
        _editableData['companies']
            ?.map((row) => row['id']?.toString() ?? '')
            .where((id) => id.isNotEmpty)
            .toSet() ??
        {};
    final allAvailableCompanyIDs = {
      ..._existingPrimaryKeys['companies'] ?? {},
      ...allCompanyIDsInFile,
    };

    _editableData.forEach((tableName, rows) {
      if (rows.isEmpty) return;
      _brokenRelationCells.putIfAbsent(tableName, () => {});

      for (int i = 0; i < rows.length; i++) {
        final row = rows[i];
        _brokenRelationCells[tableName]!.putIfAbsent(i, () => {});

        final relationsToCheck = <String, Set<String>>{
          'company_id': allAvailableCompanyIDs,
          'vendor_id': allAvailableCompanyIDs,
          'created_by': allAvailableCompanyIDs,
          'vendor': allAvailableCompanyIDs,
        };

        relationsToCheck.forEach((colName, validKeys) {
          if (row.containsKey(colName)) {
            final fk = row[colName]?.toString() ?? '';
            if (fk.isNotEmpty && !validKeys.contains(fk)) {
              _brokenRelationCells[tableName]![i]!.add(colName);
            }
          }
        });
      }
    });
  }

  void _revalidateOnDataChange() {
    setState(() {
      _validateDuplicates();
      _validateBrokenRelations();
    });
  }

  void _addRow(String tableName) {
    setState(() {
      final columns = _editableData[tableName]!.isNotEmpty
          ? _editableData[tableName]!.first.keys.toList()
          : (_templateColumns[tableName.toLowerCase()] ?? []);
      final newRow = {for (var col in columns) col: ''};
      _editableData[tableName]!.add(newRow);
      _revalidateOnDataChange();
    });
  }

  void _deleteRow(String tableName, int index) {
    setState(() {
      _editableData[tableName]!.removeAt(index);
      _revalidateOnDataChange();
    });
  }

  void _deleteColumn(String tableName, String columnName) {
    setState(() {
      for (var row in _editableData[tableName]!) {
        row.remove(columnName);
      }
      _revalidateOnDataChange();
    });
  }

  void _renameColumn(String tableName, String oldName, String newName) {
    if (newName.isNotEmpty && newName != oldName) {
      setState(() {
        for (var row in _editableData[tableName]!) {
          final value = row.remove(oldName);
          row[newName] = value;
        }
        _revalidateOnDataChange();
      });
    }
  }

  void _addNewColumn(String tableName, String newName) {
    if (newName.isNotEmpty) {
      setState(() {
        for (var row in _editableData[tableName]!) {
          row[newName] = '';
        }
        _revalidateOnDataChange();
      });
    }
  }

  Future<void> _saveToDatabase() async {
    if (!widget.isCustomTemplate) {
      final hasDuplicates = _duplicateRows.values.any((s) => s.isNotEmpty);
      if (hasDuplicates) {
        _showErrorSnackBar(
          'Data duplikat tidak bisa disimpan. Harap perbaiki.',
        );
        return;
      }
      final hasBrokenRelations = _brokenRelationCells.values.any(
        (map) => map.values.any((set) => set.isNotEmpty),
      );
      if (hasBrokenRelations) {
        _showErrorSnackBar(
          'Masih ada relasi data yang belum valid (ditandai merah). Harap perbaiki.',
        );
        return;
      }
    }

    final confirm = await showModalBottomSheet<bool>(
      context: context,
      isScrollControlled: true,
      backgroundColor: Colors.white,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(16)),
      ),
      builder: (ctx) => const ConfirmImportBottomSheet(
        title: 'Konfirmasi Impor',
        content:
            'Data akan ditambahkan atau diperbarui di database. Lanjutkan?',
      ),
    );
    if (confirm != true) return;

    // --- [LOG 1] TAMPILKAN DATA MENTAH SEBELUM DIPROSES ---
    debugPrint('--- DEBUG LOG: DATA TO BE IMPORTED ---');
    debugPrint(const JsonEncoder.withIndent('  ').convert(_editableData));
    debugPrint('-----------------------------------------');
    // ----------------------------------------------------

    final prefs = await SharedPreferences.getInstance();
    final String? loggedInUsername = prefs.getString('loggedInUsername');

    showDialog(
      context: context,
      barrierDismissible: false,
      builder: (_) => ImportProgressDialog(
        progress: _progressNotifier,
        status: _statusNotifier,
      ),
    );

    try {
      String resultMessage;
      if (widget.isCustomTemplate) {
        resultMessage = await DatabaseHelper.instance.importFromCustomTemplate(
          data: _editableData,
          onProgress: (p, m) {
            _progressNotifier.value = p;
            _statusNotifier.value = m;
          },
          loggedInUsername: loggedInUsername,
        );
      } else {
        await DatabaseHelper.instance.importData(_editableData, (p, m) {
          _progressNotifier.value = p;
          _statusNotifier.value = m;
        });
        resultMessage = "Data berhasil diimpor! 🎉";
      }

      // --- [LOG 2] TAMPILKAN PESAN HASIL DARI DATABASE HELPER ---
      debugPrint('--- DEBUG LOG: RESULT MESSAGE RECEIVED ---');
      debugPrint(resultMessage);
      debugPrint('---------------------------------------------');
      // --------------------------------------------------------

      if (mounted) {
        Navigator.of(context).pop(); // pop progress dialog
        Navigator.of(context).pop(true); // pop review screen
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
            content: Text(resultMessage),
            backgroundColor:
                resultMessage.toLowerCase().contains("gagal") ||
                    resultMessage.toLowerCase().contains("error")
                ? AppColors.red
                : AppColors.schneiderGreen,
            behavior: SnackBarBehavior.floating,
          ),
        );
      }
    } catch (e) {
      if (mounted) {
        Navigator.of(context).pop();
        _showErrorSnackBar('Gagal menyimpan data: $e');
      }
    }
  }

  void _showErrorSnackBar(String message) {
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(
        content: Text(message),
        backgroundColor: AppColors.red,
        behavior: SnackBarBehavior.floating,
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    if (_isLoading) {
      return const Scaffold(
        backgroundColor: AppColors.white,
        body: Center(
          child: Column(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              CircularProgressIndicator(color: AppColors.schneiderGreen),
              SizedBox(height: 16),
              Text(
                "Memvalidasi data...",
                style: TextStyle(color: AppColors.gray),
              ),
            ],
          ),
        ),
      );
    }
    final tableNames = _editableData.keys.toList();
    return DefaultTabController(
      length: tableNames.length,
      child: Scaffold(
        backgroundColor: AppColors.white,
        appBar: AppBar(
          scrolledUnderElevation: 0,
          backgroundColor: AppColors.white,
          surfaceTintColor: AppColors.white,
          title: const Text(
            'Tinjau Data Impor',
            style: TextStyle(
              color: AppColors.black,
              fontSize: 24,
              fontWeight: FontWeight.w400,
            ),
          ),
          bottom: PreferredSize(
            preferredSize: const Size.fromHeight(50),
            child: Align(
              alignment: Alignment.centerLeft,
              child: TabBar(
                isScrollable: true,
                labelColor: AppColors.black,
                unselectedLabelColor: AppColors.gray,
                indicatorColor: AppColors.schneiderGreen,
                indicatorWeight: 2,
                tabAlignment: TabAlignment.start,
                padding: const EdgeInsets.symmetric(horizontal: 20),
                indicatorSize: TabBarIndicatorSize.label,
                overlayColor: MaterialStateProperty.all(Colors.transparent),
                dividerColor: Colors.transparent,
                labelStyle: const TextStyle(
                  fontWeight: FontWeight.w500,
                  fontFamily: 'Lexend',
                  fontSize: 12,
                ),
                unselectedLabelStyle: const TextStyle(
                  fontWeight: FontWeight.w400,
                  fontFamily: 'Lexend',
                  fontSize: 12,
                ),
                tabs: tableNames.map(_buildTabWithIndicator).toList(),
              ),
            ),
          ),
        ),
        body: TabBarView(
          children: tableNames
              .map((name) => _buildDataTable(name, _editableData[name]!))
              .toList(),
        ),
        bottomNavigationBar: Container(
          padding: const EdgeInsets.fromLTRB(16, 12, 16, 24),
          decoration: const BoxDecoration(color: AppColors.white),
          child: ElevatedButton(
            style: ElevatedButton.styleFrom(
              minimumSize: const Size(double.infinity, 52),
              shadowColor: Colors.transparent,
              backgroundColor: AppColors.schneiderGreen,
              foregroundColor: Colors.white,
              shape: RoundedRectangleBorder(
                borderRadius: BorderRadius.circular(6),
              ),
            ),
            onPressed: _saveToDatabase,
            child: const Text(
              'Simpan ke Database',
              style: TextStyle(
                fontFamily: 'Lexend',
                fontWeight: FontWeight.w400,
                fontSize: 12,
              ),
            ),
          ),
        ),
      ),
    );
  }

  String _toTitleCase(String text) {
    if (text.isEmpty) return '';
    return text
        .split(RegExp(r'[\s_]+'))
        .map((word) {
          if (word.isEmpty) return '';
          return word[0].toUpperCase() + word.substring(1).toLowerCase();
        })
        .join(' ');
  }

  Widget _buildTabWithIndicator(String tableName) {
    final hasDuplicates = (_duplicateRows[tableName]?.isNotEmpty ?? false);
    final hasWarnings =
        (_brokenRelationCells[tableName]?.values.any((s) => s.isNotEmpty) ??
        false);
    final rowCount = _editableData[tableName]?.length ?? 0;

    Color? indicatorColor;
    if (hasDuplicates) {
      indicatorColor = AppColors.red;
    } else if (hasWarnings) {
      indicatorColor = Colors.orange;
    }

    return Tab(
      child: Row(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          Text('${_toTitleCase(tableName)} ($rowCount)'),
          if (indicatorColor != null) ...[
            const SizedBox(width: 8),
            CircleAvatar(backgroundColor: indicatorColor, radius: 4),
          ],
        ],
      ),
    );
  }

  Widget _buildInfoAlert({
    required IconData icon,
    required Color color,
    required String title,
    required Widget details,
  }) {
    return Container(
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: color.withOpacity(0.08),
        border: Border(left: BorderSide(width: 4, color: color)),
      ),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Icon(icon, color: color, size: 22),
          const SizedBox(width: 12),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  title,
                  style: TextStyle(
                    color: color,
                    fontWeight: FontWeight.w400,
                    fontSize: 14,
                  ),
                ),
                const SizedBox(height: 4),
                details,
              ],
            ),
          ),
        ],
      ),
    );
  }

  String _normalizeColumnName(String name) {
    return name.toLowerCase().replaceAll(RegExp(r'[^a-z0-9]'), '');
  }

  Widget _buildColumnValidationInfoBox(String tableName) {
    if (!_editableData.containsKey(tableName)) return const SizedBox.shrink();

    final detailsStyle = TextStyle(
      fontSize: 12,
      color: Colors.black.withOpacity(0.8),
      fontWeight: FontWeight.w300,
    );

    if (_editableData[tableName]!.isEmpty) {
      return Container(
        margin: const EdgeInsets.only(bottom: 16),
        child: _buildInfoAlert(
          icon: Icons.check_circle_outlined,
          color: AppColors.schneiderGreen,
          title: "Struktur Kolom Sesuai",
          details: Text(
            "Tidak ada data untuk diimpor di tabel ini.",
            style: detailsStyle,
          ),
        ),
      );
    }

    final expectedColumnList = _templateColumns[tableName.toLowerCase()] ?? [];
    if (expectedColumnList.isEmpty) return const SizedBox.shrink();

    final actualColumns = _editableData[tableName]!.first.keys.toList();

    final normalizedExpected = expectedColumnList
        .map(_normalizeColumnName)
        .toSet();
    final normalizedActual = actualColumns.map(_normalizeColumnName).toSet();

    final missingNormalized = normalizedExpected.difference(normalizedActual);
    final unrecognizedNormalized = normalizedActual.difference(
      normalizedExpected,
    );

    final missingColumns = expectedColumnList
        .where((col) => missingNormalized.contains(_normalizeColumnName(col)))
        .toList();
    final unrecognizedColumns = actualColumns
        .where(
          (col) => unrecognizedNormalized.contains(_normalizeColumnName(col)),
        )
        .toList();

    if (missingColumns.isEmpty && unrecognizedColumns.isEmpty) {
      return Container(
        margin: const EdgeInsets.only(bottom: 16),
        child: _buildInfoAlert(
          icon: Icons.check_circle_outlined,
          color: AppColors.schneiderGreen,
          title: "Struktur Kolom Sesuai",
          details: Text(
            "Semua kolom yang diperlukan sudah ada dan dikenali.",
            style: detailsStyle,
          ),
        ),
      );
    }

    return Container(
      margin: const EdgeInsets.only(bottom: 16),
      child: _buildInfoAlert(
        icon: Icons.warning_amber_sharp,
        color: AppColors.orange,
        title: "Struktur Kolom Tidak Sesuai",
        details: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            if (missingColumns.isNotEmpty) ...[
              const Text(
                "Kolom yang hilang:",
                style: TextStyle(fontWeight: FontWeight.w500),
              ),
              Text(
                "  • ${missingColumns.join('\n  • ')}",
                style: const TextStyle(fontFamily: 'monospace', fontSize: 12),
              ),
              const Text(
                "Gunakan tombol (+) di header untuk menambahkan.",
                style: TextStyle(fontSize: 11, color: Colors.black54),
              ),
              const SizedBox(height: 8),
            ],
            if (unrecognizedColumns.isNotEmpty) ...[
              const Text(
                "Kolom tidak dikenali:",
                style: TextStyle(fontWeight: FontWeight.w500),
              ),
              Text(
                "  • ${unrecognizedColumns.join('\n  • ')}",
                style: const TextStyle(fontFamily: 'monospace', fontSize: 12),
              ),
              const Text(
                "Ganti nama atau hapus kolom ini menggunakan menu (⋮) di header.",
                style: TextStyle(fontSize: 11, color: Colors.black54),
              ),
            ],
          ],
        ),
      ),
    );
  }

  Widget _buildDataTable(String tableName, List<Map<String, dynamic>> rows) {
    final columns = rows.isNotEmpty
        ? rows.first.keys.toList()
        : (_templateColumns[tableName.toLowerCase()] ?? []);
    return SingleChildScrollView(
      padding: const EdgeInsets.fromLTRB(20, 16, 20, 20),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          if (widget.isCustomTemplate) _buildColumnValidationInfoBox(tableName),
          if (columns.isEmpty && rows.isEmpty)
            Center(child: Text('Tidak ada data untuk tabel "$tableName".'))
          else
            Container(
              decoration: BoxDecoration(
                border: Border.all(color: AppColors.grayLight),
                borderRadius: BorderRadius.circular(8),
              ),
              child: ClipRRect(
                borderRadius: BorderRadius.circular(7),
                child: SingleChildScrollView(
                  scrollDirection: Axis.horizontal,
                  child: DataTable(
                    headingRowColor: MaterialStateProperty.all(
                      AppColors.grayLight.withOpacity(0.4),
                    ),
                    headingTextStyle: const TextStyle(
                      fontWeight: FontWeight.w500,
                      fontFamily: 'Lexend',
                      color: AppColors.black,
                      fontSize: 12,
                    ),
                    dataTextStyle: const TextStyle(
                      fontWeight: FontWeight.w300,
                      fontFamily: 'Lexend',
                      color: AppColors.black,
                      fontSize: 12,
                    ),
                    columns: [
                      ...columns.map(
                        (col) => DataColumn(
                          label: _buildColumnHeader(tableName, col),
                        ),
                      ),
                      DataColumn(
                        label: IconButton(
                          icon: const Icon(
                            Icons.add,
                            color: AppColors.schneiderGreen,
                          ),
                          tooltip: 'Tambah Kolom',
                          onPressed: () => _showAddColumnBottomSheet(tableName),
                        ),
                      ),
                      const DataColumn(label: Center(child: Text('Aksi'))),
                    ],
                    rows: List.generate(rows.length, (index) {
                      final rowData = rows[index];
                      final isDuplicate =
                          (_duplicateRows[tableName]?.contains(index) ?? false);
                      final brokenCells =
                          _brokenRelationCells[tableName]?[index] ?? <String>{};

                      return DataRow(
                        key: ObjectKey(rowData),
                        color: MaterialStateProperty.resolveWith<Color?>((s) {
                          if (isDuplicate) {
                            return AppColors.red.withOpacity(0.1);
                          }
                          return null;
                        }),
                        cells: [
                          ...columns.map(
                            (colName) => DataCell(
                              _buildCellEditor(
                                tableName,
                                index,
                                colName,
                                rowData,
                                isBroken:
                                    brokenCells.contains(colName) &&
                                    !widget.isCustomTemplate,
                              ),
                            ),
                          ),
                          const DataCell(SizedBox()),
                          DataCell(
                            Center(
                              child: IconButton(
                                icon: const Icon(
                                  Icons.more_vert,
                                  color: AppColors.gray,
                                  size: 18,
                                ),
                                onPressed: () => _showRowActionsBottomSheet(
                                  tableName,
                                  index,
                                ),
                              ),
                            ),
                          ),
                        ],
                      );
                    }),
                  ),
                ),
              ),
            ),
          const SizedBox(height: 16),
          Align(
            alignment: Alignment.centerRight,
            child: OutlinedButton.icon(
              icon: const Icon(Icons.add_circle_outline, size: 18),
              label: const Text(
                'Tambah Baris',
                style: TextStyle(
                  fontFamily: 'Lexend',
                  fontWeight: FontWeight.w400,
                  fontSize: 12,
                ),
              ),
              onPressed: () => _addRow(tableName),
              style: OutlinedButton.styleFrom(
                foregroundColor: AppColors.schneiderGreen,
                side: BorderSide(color: AppColors.gray.withOpacity(0.5)),
              ),
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildCellEditor(
    String tableName,
    int rowIndex,
    String colName,
    Map<String, dynamic> rowData, {
    required bool isBroken,
  }) {
    TextStyle textStyle = TextStyle(
      fontSize: 12,
      fontWeight: FontWeight.w300,
      fontFamily: 'Lexend',
      color: isBroken ? AppColors.red : AppColors.black,
    );

    return SizedBox(
      width: 180,
      child: TextFormField(
        initialValue: rowData[colName]?.toString() ?? '',
        keyboardType:
            (colName.contains('progress') || colName.contains('percent'))
            ? const TextInputType.numberWithOptions(decimal: true)
            : TextInputType.text,
        cursorColor: AppColors.schneiderGreen,
        style: textStyle,
        decoration: InputDecoration(
          isDense: true,
          border: InputBorder.none,
          focusedBorder: const UnderlineInputBorder(
            borderSide: BorderSide(color: AppColors.schneiderGreen, width: 1.5),
          ),
          enabledBorder: isBroken
              ? const UnderlineInputBorder(
                  borderSide: BorderSide(color: AppColors.red, width: 1.0),
                )
              : const UnderlineInputBorder(
                  borderSide: BorderSide(color: Colors.transparent),
                ),
          contentPadding: const EdgeInsets.symmetric(
            vertical: 4,
            horizontal: 2,
          ),
        ),
        onChanged: (value) {
          rowData[colName] = value;
          _revalidateOnDataChange();
        },
      ),
    );
  }

  Widget _buildColumnHeader(String tableName, String columnName) {
    return Row(
      mainAxisAlignment: MainAxisAlignment.spaceBetween,
      crossAxisAlignment: CrossAxisAlignment.center,
      children: [
        Text(_toTitleCase(columnName)),
        IconButton(
          padding: EdgeInsets.zero,
          constraints: const BoxConstraints(),
          icon: const Icon(Icons.more_vert, size: 18, color: AppColors.gray),
          onPressed: () => _showColumnActionsBottomSheet(tableName, columnName),
        ),
      ],
    );
  }

  void _showColumnActionsBottomSheet(String tableName, String columnName) {
    showModalBottomSheet(
      context: context,
      backgroundColor: Colors.white,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(16)),
      ),
      builder: (context) {
        return Padding(
          padding: const EdgeInsets.fromLTRB(20, 16, 20, 24),
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
              Text(
                'Aksi untuk Kolom "${_toTitleCase(columnName)}"',
                style: const TextStyle(
                  fontSize: 20,
                  fontWeight: FontWeight.w500,
                ),
              ),
              const SizedBox(height: 16),
              _buildBottomSheetAction(
                icon: Icons.edit_outlined,
                title: 'Ganti Nama Kolom',
                onTap: () {
                  Navigator.pop(context);
                  _showRenameColumnBottomSheet(tableName, columnName);
                },
              ),
              const Divider(height: 1),
              _buildBottomSheetAction(
                icon: Icons.delete_outline,
                title: 'Hapus Kolom',
                isDestructive: true,
                onTap: () {
                  Navigator.pop(context);
                  _showDeleteColumnConfirmationBottomSheet(
                    tableName,
                    columnName,
                  );
                },
              ),
            ],
          ),
        );
      },
    );
  }

  void _showRowActionsBottomSheet(String tableName, int index) {
    final rowData = _editableData[tableName]![index];
    final isDuplicate = (_duplicateRows[tableName]?.contains(index) ?? false);
    final brokenCells = (_brokenRelationCells[tableName]?[index] ?? <String>{});
    final pkColumn = _getPkColumn(tableName);
    final pkValue = (pkColumn.isNotEmpty && rowData.containsKey(pkColumn))
        ? rowData[pkColumn]
        : 'Baris ${index + 1}';
    showModalBottomSheet(
      context: context,
      backgroundColor: Colors.white,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(16)),
      ),
      builder: (context) {
        return Padding(
          padding: const EdgeInsets.fromLTRB(20, 16, 20, 24),
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
              Text(
                'Aksi untuk Baris "$pkValue"',
                style: const TextStyle(
                  fontSize: 20,
                  fontWeight: FontWeight.w500,
                ),
                overflow: TextOverflow.ellipsis,
              ),
              const SizedBox(height: 16),
              if (widget.isCustomTemplate ||
                  (brokenCells.isEmpty && !isDuplicate))
                const Text(
                  'Tidak ada masalah terdeteksi pada baris ini.',
                  style: TextStyle(color: AppColors.gray),
                ),
              if (brokenCells.isNotEmpty)
                _buildInfoAlert(
                  icon: Icons.error_outline,
                  color: AppColors.red,
                  title: "Error: Relasi Tidak Ditemukan",
                  details: Text(
                    'ID untuk kolom: ${brokenCells.join(', ')} tidak ditemukan.',
                    style: const TextStyle(fontSize: 12),
                  ),
                ),
              if (isDuplicate) ...[
                if (brokenCells.isNotEmpty) const SizedBox(height: 8),
                _buildInfoAlert(
                  icon: Icons.error_outline,
                  color: AppColors.red,
                  title: "Error: Data Duplikat",
                  details: Text(
                    'Nilai "$pkValue" untuk kolom "$pkColumn" sudah ada dan tidak bisa ditambahkan lagi.',
                    style: const TextStyle(fontSize: 12),
                  ),
                ),
              ],
              const SizedBox(height: 16),
              const Divider(height: 1),
              _buildBottomSheetAction(
                icon: Icons.delete_outline,
                title: 'Hapus Baris',
                isDestructive: true,
                onTap: () {
                  Navigator.pop(context);
                  _deleteRow(tableName, index);
                },
              ),
            ],
          ),
        );
      },
    );
  }

  void _showRenameColumnBottomSheet(String tableName, String oldName) {
    final controller = TextEditingController(text: oldName);
    showModalBottomSheet(
      context: context,
      isScrollControlled: true,
      backgroundColor: Colors.white,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(16)),
      ),
      builder: (context) => Padding(
        padding: EdgeInsets.fromLTRB(
          20,
          16,
          20,
          MediaQuery.of(context).viewInsets.bottom + 16,
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
            const Text(
              'Ganti Nama Kolom',
              style: TextStyle(fontSize: 20, fontWeight: FontWeight.w500),
            ),
            const SizedBox(height: 16),
            TextField(
              controller: controller,
              autofocus: true,
              style: const TextStyle(fontSize: 12, fontWeight: FontWeight.w300),
              decoration: InputDecoration(
                hintText: 'Masukkan Nama Kolom Baru',
                contentPadding: const EdgeInsets.symmetric(
                  horizontal: 16,
                  vertical: 12,
                ),
                border: OutlineInputBorder(
                  borderRadius: BorderRadius.circular(8),
                  borderSide: const BorderSide(color: AppColors.grayLight),
                ),
                enabledBorder: OutlineInputBorder(
                  borderRadius: BorderRadius.circular(8),
                  borderSide: const BorderSide(color: AppColors.grayLight),
                ),
                focusedBorder: OutlineInputBorder(
                  borderRadius: BorderRadius.circular(8),
                  borderSide: const BorderSide(color: AppColors.schneiderGreen),
                ),
              ),
            ),
            const SizedBox(height: 32),
            _buildActionButtons(
              context: context,
              onSave: () {
                final newName = controller.text.trim().replaceAll(' ', '_');
                _renameColumn(tableName, oldName, newName);
                Navigator.pop(context);
              },
            ),
          ],
        ),
      ),
    );
  }

  void _showAddColumnBottomSheet(String tableName) {
    final controller = TextEditingController();
    showModalBottomSheet(
      context: context,
      isScrollControlled: true,
      backgroundColor: Colors.white,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(16)),
      ),
      builder: (context) => Padding(
        padding: EdgeInsets.fromLTRB(
          20,
          16,
          20,
          MediaQuery.of(context).viewInsets.bottom + 16,
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
            const Text(
              'Tambah Kolom Baru',
              style: TextStyle(fontSize: 20, fontWeight: FontWeight.w500),
            ),
            const SizedBox(height: 16),
            TextField(
              controller: controller,
              autofocus: true,
              style: const TextStyle(fontSize: 12, fontWeight: FontWeight.w300),
              decoration: InputDecoration(
                hintText: 'Masukkan Nama Kolom (tanpa spasi)',
                contentPadding: const EdgeInsets.symmetric(
                  horizontal: 16,
                  vertical: 12,
                ),
                border: OutlineInputBorder(
                  borderRadius: BorderRadius.circular(8),
                  borderSide: const BorderSide(color: AppColors.grayLight),
                ),
                enabledBorder: OutlineInputBorder(
                  borderRadius: BorderRadius.circular(8),
                  borderSide: const BorderSide(color: AppColors.grayLight),
                ),
                focusedBorder: OutlineInputBorder(
                  borderRadius: BorderRadius.circular(8),
                  borderSide: const BorderSide(color: AppColors.schneiderGreen),
                ),
              ),
            ),
            const SizedBox(height: 32),
            _buildActionButtons(
              context: context,
              saveLabel: "Tambah",
              onSave: () {
                final newName = controller.text.trim().replaceAll(' ', '_');
                _addNewColumn(tableName, newName);
                Navigator.pop(context);
              },
            ),
          ],
        ),
      ),
    );
  }

  void _showDeleteColumnConfirmationBottomSheet(
    String tableName,
    String columnName,
  ) {
    showModalBottomSheet(
      context: context,
      backgroundColor: Colors.white,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(16)),
      ),
      builder: (context) => Padding(
        padding: const EdgeInsets.fromLTRB(20, 16, 20, 24),
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
            const Text(
              'Hapus Kolom?',
              style: TextStyle(fontSize: 20, fontWeight: FontWeight.w500),
            ),
            const SizedBox(height: 8),
            Text(
              'Anda yakin ingin menghapus kolom "${_toTitleCase(columnName)}"? Tindakan ini tidak dapat diurungkan.',
              style: const TextStyle(color: AppColors.gray, fontSize: 14),
            ),
            const SizedBox(height: 32),
            _buildActionButtons(
              context: context,
              saveLabel: "Ya, Hapus",
              isDestructive: true,
              onSave: () {
                _deleteColumn(tableName, columnName);
                Navigator.pop(context);
              },
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildActionButtons({
    required BuildContext context,
    required VoidCallback onSave,
    String saveLabel = "Simpan",
    bool isDestructive = false,
  }) {
    return Row(
      children: [
        Expanded(
          child: OutlinedButton(
            onPressed: () => Navigator.pop(context),
            style: OutlinedButton.styleFrom(
              padding: const EdgeInsets.symmetric(vertical: 16),
              side: const BorderSide(color: AppColors.schneiderGreen),
              shape: RoundedRectangleBorder(
                borderRadius: BorderRadius.circular(6),
              ),
            ),
            child: const Text(
              "Batal",
              style: TextStyle(color: AppColors.schneiderGreen, fontSize: 12),
            ),
          ),
        ),
        const SizedBox(width: 12),
        Expanded(
          child: ElevatedButton(
            onPressed: onSave,
            style: ElevatedButton.styleFrom(
              padding: const EdgeInsets.symmetric(vertical: 16),
              backgroundColor: isDestructive
                  ? AppColors.red
                  : AppColors.schneiderGreen,
              foregroundColor: Colors.white,
              elevation: 0,
              shape: RoundedRectangleBorder(
                borderRadius: BorderRadius.circular(6),
              ),
            ),
            child: Text(saveLabel, style: const TextStyle(fontSize: 12)),
          ),
        ),
      ],
    );
  }

  Widget _buildBottomSheetAction({
    required IconData icon,
    required String title,
    required VoidCallback onTap,
    bool isDestructive = false,
  }) {
    final color = isDestructive ? AppColors.red : AppColors.black;
    return InkWell(
      onTap: onTap,
      borderRadius: BorderRadius.circular(8),
      child: Padding(
        padding: const EdgeInsets.symmetric(vertical: 12.0, horizontal: 8.0),
        child: Row(
          children: [
            Icon(icon, color: color),
            const SizedBox(width: 16),
            Text(
              title,
              style: TextStyle(
                color: color,
                fontSize: 14,
                fontWeight: FontWeight.w400,
              ),
            ),
          ],
        ),
      ),
    );
  }

  String _getPkColumn(String tableName) {
    const Map<String, String> pkMap = {
      'panels': 'no_pp',
      'companies': 'id',
      'company_accounts': 'username',
      'Panel': 'PP Panel',
      'User': 'Username',
    };
    return pkMap[tableName] ?? '';
  }
}
