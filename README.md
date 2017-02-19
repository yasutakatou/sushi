*SuSHi Toolとは*

監視ツールというか、SSHを介したジョブ伝播ツールがこのツールとなります。  
他ツールとの連携を捨て、とことんレガシーの処理の組み合わせで要件の実現と  
ボトルネック部分を省くことに焦点を当て、  
デッドロックと大規模運用に耐えられるよう実装しました。  
グルーピング機能の実装にredisは使ってますが。。  

#まとめ：　このツールは以下の特徴があります。

##１．　outboundの通信のみで監視可能  
###-　内側の通信許可不要なので強固なセキュリティ要件でも対応できる！  
##２．　go言語で作られているのでインストールはバイナリの配置のみ  
###-　ワークフレームや他のツールをインストールする前提が無く、迅速な環境デプロイ！  
##３．　リポジトリ、グループ、クラスターモードを実装  
###-　設定のリポジトリによる監視方法の切り替え迅速化、複数設定も一度に実行、冗長性も確保！  
##４．　フルAPI  
###-　全ての項目をAPIのみで監視設定可能！  
##５．　柔軟な監視設定が可能  
###-　実行命令を仕掛ける方法なのでツール内の動作のクセを意識せず監視設定できる！  

#実装方式が柔軟なため未実装ですが将来的に以下の拡張性を持っています。  

##１．　Windows版Agentに対応  
###-　go言語で作られているため実行方法の部分をWindowsのコマンドプロンプトに置き換えることでWindowsも監視可能になる  
##２．　ChatOpsに対応  
###-　フルAPIかつ、監視方法がテキストで設定できるのでBotと連携することでSlack等から監視出来るツールに  
##３．　オーケストレーションに対応  
###-　監視対象Agentに任意のコマンドが投げられるので監視以外でもインベントリ収集など構成管理機能も実装可能 
