import 'package:flutter/material.dart';
import 'package:intl/intl.dart';
import 'package:secpanel/components/import/confirm_import_bottom_sheet.dart';
import 'package:secpanel/components/import/import_progress_dialog.dart';
import 'package:secpanel/helpers/db_helper.dart';
import 'package:secpanel/models/approles.dart';
import 'package:secpanel/models/company.dart';
import 'package:secpanel/theme/colors.dart';

class ImportReviewScreen extends StatefulWidget {
  final Map<String, List<Map<String, dynamic>>> initialData;
  const ImportReviewScreen({super.key, required this.initialData});

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
    'companies': ['id', 'name', 'role'],
    'company_accounts': ['username', 'password', 'company_id'],
    'panels': [
      'no_pp',
      'no_panel',
      'no_wbs',
      'percent_progress',
      'start_date',
      'target_delivery',
      'status_busbar_pcc',
      'status_busbar_mcc',
      'status_component',
      'status_palet',
      'status_corepart',
      'ao_busbar_pcc',
      'eta_busbar_pcc',
      'ao_busbar_mcc',
      'eta_busbar_mcc',
      'ao_component',
      'eta_component',
      'created_by',
      'vendor_id',
      'is_closed',
      'closed_date',
    ],
    'busbars': ['panel_no_pp', 'vendor', 'remarks'],
    'components': ['panel_no_pp', 'vendor'],
    'palet': ['panel_no_pp', 'vendor'],
    'corepart': ['panel_no_pp', 'vendor'],
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
    _brokenRelationCells = {};
    _initializeAndValidateData();
  }

  Future<void> _initializeAndValidateData() async {
    if (mounted) setState(() => _isLoading = true);
    await _fetchExistingPrimaryKeys();
    await _fetchAllCompanies();
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
          : _templateColumns[tableName] ?? <String>[];
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
    final hasDuplicates = _duplicateRows.values.any((s) => s.isNotEmpty);
    if (hasDuplicates) {
      _showErrorSnackBar('Data duplikat tidak bisa disimpan. Harap perbaiki.');
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
            'Data yang valid akan ditambahkan atau diperbarui di database. Lanjutkan?',
      ),
    );
    if (confirm != true) return;
    showDialog(
      context: context,
      barrierDismissible: false,
      builder: (_) => ImportProgressDialog(
        progress: _progressNotifier,
        status: _statusNotifier,
      ),
    );
    try {
      await DatabaseHelper.instance.importData(_editableData, (p, m) {
        _progressNotifier.value = p;
        _statusNotifier.value = m;
      });
      if (mounted) {
        Navigator.of(context).pop();
        Navigator.of(context).pop(true);
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(
            content: Text('Data berhasil diimpor! 🎉'),
            backgroundColor: AppColors.schneiderGreen,
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
                overlayColor: WidgetStateProperty.all(Colors.transparent),
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
    final hasDuplicates = _duplicateRows[tableName]?.isNotEmpty ?? false;
    final hasWarnings =
        _brokenRelationCells[tableName]?.values.any((s) => s.isNotEmpty) ??
        false;
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
    final expectedColumns = _templateColumns[tableName]?.toSet() ?? <String>{};
    if (expectedColumns.isEmpty) return const SizedBox.shrink();
    final actualColumns = _editableData[tableName]!.first.keys.toSet();
    final missingColumns = expectedColumns.difference(actualColumns).toList();
    final unrecognizedColumns = actualColumns
        .difference(expectedColumns)
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
                "  • ${missingColumns.join('\n  • ').toLowerCase()}",
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
                "  • ${unrecognizedColumns.join('\n  • ').toLowerCase()}",
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
        : (_templateColumns[tableName] ?? []);
    return SingleChildScrollView(
      padding: const EdgeInsets.fromLTRB(20, 16, 20, 20),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          _buildColumnValidationInfoBox(tableName),
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
                    headingRowColor: WidgetStateProperty.all(
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
                          _duplicateRows[tableName]?.contains(index) ?? false;
                      final brokenCells =
                          _brokenRelationCells[tableName]?[index] ?? {};
                      return DataRow(
                        color: WidgetStateProperty.resolveWith<Color?>((s) {
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
                                isBroken: brokenCells.contains(colName),
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

    Widget buildSelectableCell({
      required String displayValue,
      required VoidCallback onTap,
      bool isError = false,
    }) {
      return SizedBox(
        width: 180,
        child: InkWell(
          onTap: onTap,
          child: Container(
            decoration: BoxDecoration(
              border: Border(
                bottom: BorderSide(
                  color: isError ? AppColors.red : Colors.transparent,
                  width: 1.0,
                ),
              ),
            ),
            padding: const EdgeInsets.symmetric(vertical: 12, horizontal: 2),
            alignment: Alignment.centerLeft,
            child: Text(
              displayValue,
              style: textStyle.copyWith(
                color: isError
                    ? AppColors.red
                    : (displayValue == 'Pilih...' ||
                              displayValue == 'Pilih Tanggal' ||
                              displayValue == 'Pilih Role' ||
                              displayValue.isEmpty
                          ? AppColors.gray
                          : AppColors.black),
              ),
              overflow: TextOverflow.ellipsis,
            ),
          ),
        ),
      );
    }

    if (colName.contains('_date') || colName.contains('delivery')) {
      DateTime? currentDate;
      final dateValue = rowData[colName];
      if (dateValue is String && dateValue.isNotEmpty) {
        currentDate = DateTime.tryParse(dateValue);
      }
      return buildSelectableCell(
        isError: isBroken,
        displayValue: currentDate != null
            ? DateFormat('d MMM yyyy').format(currentDate)
            : 'Pilih Tanggal',
        onTap: () async {
          final DateTime? pickedDate = await showDatePicker(
            context: context,
            initialDate: currentDate ?? DateTime.now(),
            firstDate: DateTime(2000),
            lastDate: DateTime(2101),
            builder: (context, child) {
              return Theme(
                data: ThemeData.light().copyWith(
                  colorScheme: const ColorScheme.light(
                    primary: AppColors.schneiderGreen,
                  ),
                ),
                child: child!,
              );
            },
          );
          if (pickedDate != null) {
            setState(() {
              rowData[colName] = DateFormat(
                "yyyy-MM-dd'T'HH:mm:ss",
              ).format(pickedDate);
              _revalidateOnDataChange();
            });
          }
        },
      );
    }

    if (colName.startsWith('status_')) {
      List<String> options;
      if (colName.contains('busbar')) {
        options = ["On Progress", "Siap 100%", "Close", "Red Block"];
      } else if (colName.contains('component')) {
        options = ["Open", "On Progress", "Done"];
      } else {
        options = ["Open", "Close"];
      }
      return buildSelectableCell(
        isError: isBroken,
        displayValue: rowData[colName] != null && rowData[colName].isNotEmpty
            ? rowData[colName]
            : 'Pilih...',
        onTap: () async {
          final selectedStatus = await _showSimpleSelectionSheet(
            context: context,
            title: "Pilih Status",
            options: options,
          );
          if (selectedStatus != null) {
            setState(() {
              rowData[colName] = selectedStatus;
              _revalidateOnDataChange();
            });
          }
        },
      );
    }

    if (colName == 'company_id' ||
        colName == 'vendor_id' ||
        colName == 'created_by' ||
        colName == 'vendor') {
      String currentId = rowData[colName]?.toString() ?? '';
      String displayName = _allCompanies
          .firstWhere(
            (c) => c.id == currentId,
            orElse: () => Company(id: '', name: 'Pilih...', role: AppRole.k3),
          )
          .name;
      return buildSelectableCell(
        isError: isBroken,
        displayValue: displayName,
        onTap: () async {
          final selectedCompany = await _showCompanySelectionSheet(
            context: context,
          );
          if (selectedCompany != null) {
            setState(() {
              if (!_allCompanies.any((c) => c.id == selectedCompany.id)) {
                _allCompanies.add(selectedCompany);
              }
              rowData[colName] = selectedCompany.id;
              _revalidateOnDataChange();
            });
          }
        },
      );
    }

    if (colName == 'role') {
      String currentRole = rowData[colName]?.toString() ?? '';
      return buildSelectableCell(
        isError: isBroken,
        displayValue: currentRole.isNotEmpty
            ? (currentRole[0].toUpperCase() + currentRole.substring(1))
            : 'Pilih Role',
        onTap: () async {
          final selectedRole = await _showRoleSelectionSheet(context: context);
          if (selectedRole != null) {
            setState(() {
              rowData[colName] = selectedRole.name;
              _revalidateOnDataChange();
            });
          }
        },
      );
    }

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
    final isDuplicate = _duplicateRows[tableName]?.contains(index) ?? false;
    final brokenCells = _brokenRelationCells[tableName]?[index] ?? {};
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
              ),
              const SizedBox(height: 16),
              if (brokenCells.isEmpty && !isDuplicate)
                const Text(
                  'Tidak ada masalah pada baris ini.',
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

  Widget _buildColumnHeader(String tableName, String columnName) {
    final expectedColumns = _templateColumns[tableName]?.toSet() ?? <String>{};
    final bool isUnrecognized = !expectedColumns.contains(columnName);

    return Row(
      mainAxisAlignment: MainAxisAlignment.spaceBetween,
      crossAxisAlignment: CrossAxisAlignment.center,
      children: [
        Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            Text(_toTitleCase(columnName)),
            Text(
              columnName,
              style: TextStyle(
                color: isUnrecognized ? AppColors.red : AppColors.gray,
                fontSize: 10,
                fontWeight: FontWeight.w300,
              ),
            ),
          ],
        ),
        IconButton(
          padding: EdgeInsets.zero,
          constraints: const BoxConstraints(),
          icon: const Icon(Icons.more_vert, size: 18, color: AppColors.gray),
          onPressed: () => _showColumnActionsBottomSheet(tableName, columnName),
        ),
      ],
    );
  }

  String _getPkColumn(String tableName) {
    const Map<String, String> pkMap = {
      'panels': 'no_pp',
      'companies': 'id',
      'company_accounts': 'username',
    };
    return pkMap[tableName] ?? '';
  }

  String _getBrokenRelationMessage(String tableName) {
    return 'Peringatan: ID pada kolom ini tidak ditemukan di data yang ada.';
  }

  Future<Company?> _showCompanySelectionSheet({
    required BuildContext context,
  }) async {
    return await showModalBottomSheet<Company>(
      context: context,
      isScrollControlled: true,
      backgroundColor: Colors.white,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(16)),
      ),
      builder: (_) => _CompanySelectionSheet(
        companies: _allCompanies,
        onAddNew: () async {
          Navigator.pop(context);
          return await _showAddNewCompanySheet(context: context);
        },
      ),
    );
  }

  Future<AppRole?> _showRoleSelectionSheet({
    required BuildContext context,
  }) async {
    return await showModalBottomSheet<AppRole>(
      context: context,
      isScrollControlled: true,
      backgroundColor: Colors.white,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(16)),
      ),
      builder: (_) => const _RoleSelectionSheet(),
    );
  }

  Future<String?> _showSimpleSelectionSheet({
    required BuildContext context,
    required String title,
    required List<String> options,
  }) async {
    return await showModalBottomSheet<String>(
      context: context,
      isScrollControlled: true,
      backgroundColor: Colors.white,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(16)),
      ),
      builder: (_) => _SimpleSelectionSheet(title: title, options: options),
    );
  }

  Future<Company?> _showAddNewCompanySheet({
    required BuildContext context,
  }) async {
    final newCompanyData = await showModalBottomSheet<Map<String, dynamic>>(
      context: context,
      isScrollControlled: true,
      backgroundColor: Colors.white,
      builder: (_) => const _AddNewCompanyRoleSheet(),
    );

    if (newCompanyData != null) {
      final String newName = newCompanyData['name'];
      final AppRole newRole = newCompanyData['role'];
      final newId = newName.toLowerCase().replaceAll(RegExp(r'\\s+'), '_');

      final newCompany = Company(id: newId, name: newName, role: newRole);
      return newCompany;
    }
    return null;
  }
}

class _CompanySelectionSheet extends StatelessWidget {
  final List<Company> companies;
  final Future<Company?> Function() onAddNew;

  const _CompanySelectionSheet({
    required this.companies,
    required this.onAddNew,
  });

  @override
  Widget build(BuildContext context) {
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
          const Text(
            "Pilih Perusahaan",
            style: TextStyle(fontSize: 20, fontWeight: FontWeight.w500),
          ),
          const SizedBox(height: 16),
          Flexible(
            child: SingleChildScrollView(
              child: Wrap(
                spacing: 8,
                runSpacing: 12,
                children: [
                  ...companies.map((company) {
                    return _buildCompanyOptionButton(
                      context: context,
                      name: company.name,
                      role: company.role.name,
                      onTap: () => Navigator.pop(context, company),
                    );
                  }),
                  _buildOtherButton(
                    onTap: () async {
                      final newCompany = await onAddNew();
                      if (newCompany != null && context.mounted) {
                        Navigator.pop(context, newCompany);
                      }
                    },
                  ),
                ],
              ),
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildCompanyOptionButton({
    required BuildContext context,
    required String name,
    required String role,
    required VoidCallback onTap,
  }) {
    return GestureDetector(
      onTap: onTap,
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
        decoration: BoxDecoration(
          color: Colors.white,
          border: Border.all(color: AppColors.grayLight),
          borderRadius: BorderRadius.circular(8),
        ),
        child: Row(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.center,
          children: [
            Text(
              name,
              style: const TextStyle(
                fontWeight: FontWeight.w400,
                fontSize: 12,
                color: AppColors.black,
              ),
            ),
            const SizedBox(width: 8),
            Chip(
              label: Text(
                role[0].toUpperCase() + role.substring(1),
                style: const TextStyle(fontSize: 10, color: AppColors.gray),
              ),
              backgroundColor: AppColors.grayLight,
              padding: EdgeInsets.zero,
              labelPadding: const EdgeInsets.symmetric(horizontal: 6),
              visualDensity: VisualDensity.compact,
              side: BorderSide.none,
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildOtherButton({required VoidCallback onTap}) {
    return GestureDetector(
      onTap: onTap,
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
        decoration: BoxDecoration(
          color: Colors.white,
          border: Border.all(color: AppColors.grayLight),
          borderRadius: BorderRadius.circular(8),
        ),
        child: const Text(
          "Lainnya...",
          style: TextStyle(
            fontWeight: FontWeight.w400,
            fontSize: 12,
            color: AppColors.gray,
          ),
        ),
      ),
    );
  }
}

class _AddNewCompanyRoleSheet extends StatefulWidget {
  const _AddNewCompanyRoleSheet();
  @override
  State<_AddNewCompanyRoleSheet> createState() =>
      _AddNewCompanyRoleSheetState();
}

class _AddNewCompanyRoleSheetState extends State<_AddNewCompanyRoleSheet> {
  final _formKey = GlobalKey<FormState>();
  final _nameController = TextEditingController();
  AppRole _selectedRole = AppRole.k3;

  @override
  void dispose() {
    _nameController.dispose();
    super.dispose();
  }

  void _save() {
    if (_formKey.currentState!.validate()) {
      Navigator.pop(context, {
        'name': _nameController.text.trim(),
        'role': _selectedRole,
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    return SingleChildScrollView(
      padding: EdgeInsets.fromLTRB(
        20,
        16,
        20,
        MediaQuery.of(context).viewInsets.bottom + 16,
      ),
      child: Form(
        key: _formKey,
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
              "Tambah Perusahaan Baru",
              style: TextStyle(fontSize: 20, fontWeight: FontWeight.w500),
            ),
            const SizedBox(height: 24),
            Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                const Text(
                  'Nama Perusahaan',
                  style: TextStyle(fontSize: 14, fontWeight: FontWeight.w400),
                ),
                const SizedBox(height: 12),
                TextFormField(
                  cursorColor: AppColors.schneiderGreen,
                  controller: _nameController,
                  autofocus: true,
                  style: const TextStyle(
                    fontSize: 12,
                    fontWeight: FontWeight.w300,
                    color: AppColors.black,
                  ),
                  validator: (v) => (v == null || v.isEmpty)
                      ? 'Nama tidak boleh kosong'
                      : null,
                  decoration: InputDecoration(
                    fillColor: AppColors.white,
                    filled: true,
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
                      borderSide: const BorderSide(
                        color: AppColors.schneiderGreen,
                      ),
                    ),
                  ),
                ),
              ],
            ),
            const SizedBox(height: 24),
            _buildRoleSelector(),
            const SizedBox(height: 32),
            Row(
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
                      style: TextStyle(
                        color: AppColors.schneiderGreen,
                        fontSize: 12,
                      ),
                    ),
                  ),
                ),
                const SizedBox(width: 12),
                Expanded(
                  child: ElevatedButton(
                    onPressed: _save,
                    style: ElevatedButton.styleFrom(
                      padding: const EdgeInsets.symmetric(vertical: 16),
                      backgroundColor: AppColors.schneiderGreen,
                      foregroundColor: Colors.white,
                      elevation: 0,
                      shape: RoundedRectangleBorder(
                        borderRadius: BorderRadius.circular(6),
                      ),
                    ),
                    child: const Text("Simpan", style: TextStyle(fontSize: 12)),
                  ),
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildRoleSelector() {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        const Text(
          'Role',
          style: TextStyle(fontSize: 14, fontWeight: FontWeight.w400),
        ),
        const SizedBox(height: 12),
        Wrap(
          spacing: 8,
          runSpacing: 12,
          children: AppRole.values.map((role) {
            if (role == AppRole.admin) return const SizedBox.shrink();
            return _buildOptionButton(
              label: role.name[0].toUpperCase() + role.name.substring(1),
              selected: _selectedRole == role,
              onTap: () => setState(() => _selectedRole = role),
            );
          }).toList(),
        ),
      ],
    );
  }

  Widget _buildOptionButton({
    required String label,
    required bool selected,
    required VoidCallback onTap,
  }) {
    final Color borderColor = selected
        ? AppColors.schneiderGreen
        : AppColors.grayLight;
    final Color color = selected
        ? AppColors.schneiderGreen.withOpacity(0.08)
        : Colors.white;

    return GestureDetector(
      onTap: onTap,
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
        decoration: BoxDecoration(
          color: color,
          border: Border.all(color: borderColor),
          borderRadius: BorderRadius.circular(8),
        ),
        child: Text(
          label,
          style: const TextStyle(
            fontWeight: FontWeight.w400,
            fontSize: 12,
            color: AppColors.black,
          ),
        ),
      ),
    );
  }
}

class _RoleSelectionSheet extends StatelessWidget {
  const _RoleSelectionSheet();

  @override
  Widget build(BuildContext context) {
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
          const Text(
            "Pilih Role",
            style: TextStyle(fontSize: 20, fontWeight: FontWeight.w500),
          ),
          const SizedBox(height: 16),
          Flexible(
            child: SingleChildScrollView(
              child: Wrap(
                spacing: 8,
                runSpacing: 12,
                children: AppRole.values
                    .where((role) => role != AppRole.admin)
                    .map((role) {
                      return GestureDetector(
                        onTap: () => Navigator.pop(context, role),
                        child: Container(
                          padding: const EdgeInsets.symmetric(
                            horizontal: 16,
                            vertical: 12,
                          ),
                          decoration: BoxDecoration(
                            color: Colors.white,
                            border: Border.all(color: AppColors.grayLight),
                            borderRadius: BorderRadius.circular(8),
                          ),
                          child: Text(
                            role.name[0].toUpperCase() + role.name.substring(1),
                            style: const TextStyle(
                              fontWeight: FontWeight.w400,
                              fontSize: 12,
                              color: AppColors.black,
                            ),
                          ),
                        ),
                      );
                    })
                    .toList(),
              ),
            ),
          ),
        ],
      ),
    );
  }
}

class _SimpleSelectionSheet extends StatelessWidget {
  final String title;
  final List<String> options;

  const _SimpleSelectionSheet({required this.title, required this.options});

  @override
  Widget build(BuildContext context) {
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
            title,
            style: const TextStyle(fontSize: 20, fontWeight: FontWeight.w500),
          ),
          const SizedBox(height: 16),
          Flexible(
            child: SingleChildScrollView(
              child: Wrap(
                spacing: 8,
                runSpacing: 12,
                children: options.map((option) {
                  return GestureDetector(
                    onTap: () => Navigator.pop(context, option),
                    child: Container(
                      padding: const EdgeInsets.symmetric(
                        horizontal: 16,
                        vertical: 12,
                      ),
                      decoration: BoxDecoration(
                        color: Colors.white,
                        border: Border.all(color: AppColors.grayLight),
                        borderRadius: BorderRadius.circular(8),
                      ),
                      child: Text(
                        option,
                        style: const TextStyle(
                          fontWeight: FontWeight.w400,
                          fontSize: 12,
                          color: AppColors.black,
                        ),
                      ),
                    ),
                  );
                }).toList(),
              ),
            ),
          ),
        ],
      ),
    );
  }
}
